package server

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skywave-chat/skywave/pkg/protocol"
)

// Hub maintains active clients, rooms, and routes packets.
type Hub struct {
	ServerName     string
	ServerPassword string
	MOTD           string
	MaxHistory     int

	clients    map[*Client]bool
	clientsByNick map[string]*Client
	rooms      map[string]*Room
	mu         sync.RWMutex

	register   chan *Client
	unregister chan *Client
}

// NewHub initializes a new Hub.
func NewHub(serverName, serverPassword, motd string, maxHistory int) *Hub {
	h := &Hub{
		ServerName:     serverName,
		ServerPassword: serverPassword,
		MOTD:           motd,
		MaxHistory:     maxHistory,
		clients:        make(map[*Client]bool),
		clientsByNick:  make(map[string]*Client),
		rooms:          make(map[string]*Room),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
	}

	// Create standard starter channels
	h.rooms["#general"] = NewRoom("#general", "Welcome to Skywave! The open global wave.", "", maxHistory)
	h.rooms["#lounge"] = NewRoom("#lounge", "Casual chat, hangouts & chill vibe.", "", maxHistory)
	h.rooms["#tech"] = NewRoom("#tech", "Code, terminals, Linux, hardware & security.", "", maxHistory)

	return h
}

// Run starts the Hub main event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[Connect] Client connected (ID: %s)", client.ID)

		case client := <-h.unregister:
			h.handleDisconnect(client)
		}
	}
}

// Register adds a new client.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister disconnects a client.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// handleDisconnect cleans up client state across rooms and nick map.
func (h *Hub) handleDisconnect(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	if client.Nickname != "" {
		delete(h.clientsByNick, strings.ToLower(client.Nickname))
	}

	// Remove from current room
	if client.CurrentRoom != "" {
		if room, exists := h.rooms[client.CurrentRoom]; exists {
			room.RemoveMember(client)
			room.Broadcast(protocol.NewPacket(protocol.TypeSystemMsg, protocol.SystemMsgPayload{
				Room:    room.Name,
				Content: fmt.Sprintf("[-] %s has left the frequency.", client.Nickname),
				Level:   "info",
			}))
			h.broadcastUserList(room)
		}
	}

	log.Printf("[Disconnect] Client disconnected (Nick: %s, ID: %s)", client.Nickname, client.ID)
}

// HandlePacket processes incoming packet from a client.
func (h *Hub) HandlePacket(c *Client, p protocol.Packet) {
	switch p.Type {
	case protocol.TypeAuth:
		h.handleAuth(c, p.Payload)
	case protocol.TypeChatMsg:
		h.handleChatMsg(c, p.Payload)
	case protocol.TypeDirectMsg:
		h.handleDirectMsg(c, p.Payload)
	case protocol.TypeJoinRoom:
		h.handleJoinRoom(c, p.Payload)
	case protocol.TypeLeaveRoom:
		h.handleLeaveRoom(c)
	case protocol.TypeSetNick:
		h.handleSetNick(c, p.Payload)
	case protocol.TypeSetTopic:
		h.handleSetTopic(c, p.Payload)
	case protocol.TypeTyping:
		h.handleTyping(c, p.Payload)
	case protocol.TypePing:
		c.Send(protocol.NewPacket(protocol.TypePong, "pong"))
	default:
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "UNKNOWN_TYPE",
			Message: fmt.Sprintf("Packet type '%s' is not recognized", p.Type),
		}))
	}
}

// handleAuth handles login / handshake.
func (h *Hub) handleAuth(c *Client, payload any) {
	raw, _ := json.Marshal(payload)
	var auth protocol.AuthPayload
	if err := json.Unmarshal(raw, &auth); err != nil {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "AUTH_ERROR",
			Message: "Invalid auth payload",
		}))
		return
	}

	// Check server password if server is private
	if h.ServerPassword != "" && auth.ServerPassword != h.ServerPassword {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "AUTH_FAILED",
			Message: "Incorrect server password",
		}))
		c.Close()
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Validate nickname
	desiredNick := strings.TrimSpace(auth.Nickname)
	if desiredNick == "" {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "INVALID_NICK",
			Message: "Callsign is required to enter the wave",
		}))
		return
	}

	cleanNick, err := protocol.ValidateNickname(desiredNick)
	if err != nil {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "INVALID_NICK",
			Message: err.Error(),
		}))
		return
	}

	// Strictly reject duplicate callsigns — no automatic _1, _2 suffixing!
	lower := strings.ToLower(cleanNick)
	if _, exists := h.clientsByNick[lower]; exists {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "NICK_IN_USE",
			Message: fmt.Sprintf("Callsign '%s' is already taken by another soul. Choose another name.", cleanNick),
		}))
		return
	}

	c.Nickname = cleanNick
	h.clientsByNick[lower] = c

	// Prepare room summaries
	roomStatuses := make([]protocol.RoomStatus, 0, len(h.rooms))
	for _, r := range h.rooms {
		roomStatuses = append(roomStatuses, r.Status())
	}

	// Send AuthAck
	c.Send(protocol.NewPacket(protocol.TypeAuthAck, protocol.AuthAckPayload{
		SessionID:  c.ID,
		Nickname:   c.Nickname,
		ServerName: h.ServerName,
		MOTD:       h.MOTD,
		Rooms:      roomStatuses,
	}))

	// Auto-join #general
	h.joinRoomInternal(c, "#general", "")
}

// handleChatMsg broadcasts a message inside the client's current room.
func (h *Hub) handleChatMsg(c *Client, payload any) {
	if c.Nickname == "" || c.CurrentRoom == "" {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "NOT_IN_ROOM",
			Message: "You must authenticate and join a room before sending messages",
		}))
		return
	}

	raw, _ := json.Marshal(payload)
	var chat protocol.ChatMsgPayload
	if err := json.Unmarshal(raw, &chat); err != nil {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "INVALID_MESSAGE",
			Message: "Failed to parse message payload",
		}))
		return
	}

	// Sanitize content against terminal escape injection attacks
	content := protocol.SanitizeText(chat.Content)
	if strings.TrimSpace(content) == "" {
		return
	}

	h.mu.RLock()
	room, exists := h.rooms[c.CurrentRoom]
	h.mu.RUnlock()

	if !exists {
		return
	}

	msg := protocol.ChatMsgPayload{
		ID:          uuid.New().String()[:12],
		Room:        room.Name,
		Sender:      c.Nickname,
		Content:     content,
		IsEncrypted: chat.IsEncrypted,
		Timestamp:   time.Now().UnixMilli(),
	}

	// Save to ephemeral in-memory buffer
	room.AddMessage(msg)

	// Broadcast to all room subscribers
	room.Broadcast(protocol.NewPacket(protocol.TypeChatMsg, msg))
}

// handleDirectMsg routes a private 1-on-1 direct message.
func (h *Hub) handleDirectMsg(c *Client, payload any) {
	raw, _ := json.Marshal(payload)
	var dm protocol.DirectMsgPayload
	if err := json.Unmarshal(raw, &dm); err != nil {
		return
	}

	content := protocol.SanitizeText(dm.Content)
	if strings.TrimSpace(content) == "" {
		return
	}

	h.mu.RLock()
	targetClient, exists := h.clientsByNick[strings.ToLower(dm.Recipient)]
	h.mu.RUnlock()

	if !exists {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "USER_NOT_FOUND",
			Message: fmt.Sprintf("User '%s' is not online", dm.Recipient),
		}))
		return
	}

	packetData := protocol.DirectMsgPayload{
		ID:          uuid.New().String()[:12],
		Sender:      c.Nickname,
		Recipient:   targetClient.Nickname,
		Content:     content,
		IsEncrypted: dm.IsEncrypted,
		Timestamp:   time.Now().UnixMilli(),
	}

	pkt := protocol.NewPacket(protocol.TypeDirectMsg, packetData)
	targetClient.Send(pkt)
	c.Send(pkt) // echo back to sender
}

// handleJoinRoom handles joining an existing room or creating a new room.
func (h *Hub) handleJoinRoom(c *Client, payload any) {
	raw, _ := json.Marshal(payload)
	var join protocol.JoinRoomPayload
	if err := json.Unmarshal(raw, &join); err != nil {
		return
	}

	normalizedName, err := protocol.NormalizeRoomName(join.RoomName)
	if err != nil {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "INVALID_ROOM_NAME",
			Message: err.Error(),
		}))
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.joinRoomInternal(c, normalizedName, join.Password)
}

// joinRoomInternal (must be called with h.mu held)
func (h *Hub) joinRoomInternal(c *Client, roomName string, password string) {
	// If currently in a room, leave it first
	if c.CurrentRoom != "" {
		if oldRoom, exists := h.rooms[c.CurrentRoom]; exists {
			oldRoom.RemoveMember(c)
			oldRoom.Broadcast(protocol.NewPacket(protocol.TypeSystemMsg, protocol.SystemMsgPayload{
				Room:    oldRoom.Name,
				Content: fmt.Sprintf("[-] %s switched frequencies.", c.Nickname),
				Level:   "info",
			}))
			h.broadcastUserList(oldRoom)
		}
	}

	room, exists := h.rooms[roomName]
	if !exists {
		// Create new room dynamically
		topic := "Custom frequency created by " + c.Nickname
		room = NewRoom(roomName, topic, password, h.MaxHistory)
		h.rooms[roomName] = room
		log.Printf("[Room] New room created: %s (Private: %t)", roomName, room.IsPrivate)
	} else {
		// Room exists, verify password if private
		if !room.Authenticate(password) {
			c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
				Code:    "ROOM_ACCESS_DENIED",
				Message: fmt.Sprintf("Incorrect password for private room %s", roomName),
			}))
			return
		}
	}

	c.CurrentRoom = room.Name
	room.AddMember(c)

	// Send room history catchup
	history := room.GetHistory()
	for _, histMsg := range history {
		c.Send(protocol.NewPacket(protocol.TypeChatMsg, histMsg))
	}

	// Announce join
	room.Broadcast(protocol.NewPacket(protocol.TypeSystemMsg, protocol.SystemMsgPayload{
		Room:    room.Name,
		Content: fmt.Sprintf("[+] %s tuned into %s", c.Nickname, room.Name),
		Level:   "success",
	}))

	h.broadcastUserList(room)
	h.broadcastRoomList()
}

// handleLeaveRoom leaves the current room and returns to #general.
func (h *Hub) handleLeaveRoom(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if c.CurrentRoom == "#general" {
		return // already in base room
	}

	h.joinRoomInternal(c, "#general", "")
}

// handleSetNick updates user's nickname.
func (h *Hub) handleSetNick(c *Client, payload any) {
	raw, _ := json.Marshal(payload)
	var nickReq protocol.SetNickPayload
	if err := json.Unmarshal(raw, &nickReq); err != nil {
		return
	}

	cleanNick, err := protocol.ValidateNickname(nickReq.NewNickname)
	if err != nil {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "INVALID_NICK",
			Message: err.Error(),
		}))
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	lower := strings.ToLower(cleanNick)
	if _, exists := h.clientsByNick[lower]; exists && lower != strings.ToLower(c.Nickname) {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "NICK_IN_USE",
			Message: fmt.Sprintf("Nickname '%s' is already in use", cleanNick),
		}))
		return
	}

	oldNick := c.Nickname
	delete(h.clientsByNick, strings.ToLower(oldNick))
	c.Nickname = cleanNick
	h.clientsByNick[lower] = c

	c.Send(protocol.NewPacket(protocol.TypeNickAck, protocol.NickAckPayload{
		OldNickname: oldNick,
		NewNickname: cleanNick,
	}))

	c.Send(protocol.NewPacket(protocol.TypeSystemMsg, protocol.SystemMsgPayload{
		Content: fmt.Sprintf("Nickname changed to %s", cleanNick),
		Level:   "success",
	}))

	if c.CurrentRoom != "" {
		if room, exists := h.rooms[c.CurrentRoom]; exists {
			room.Broadcast(protocol.NewPacket(protocol.TypeSystemMsg, protocol.SystemMsgPayload{
				Room:    room.Name,
				Content: fmt.Sprintf("[*] %s is now known as %s", oldNick, cleanNick),
				Level:   "info",
			}))
			h.broadcastUserList(room)
		}
	}
}

// handleSetTopic updates a room topic and announces it.
func (h *Hub) handleSetTopic(c *Client, payload any) {
	raw, _ := json.Marshal(payload)
	var t protocol.TopicPayload
	if err := json.Unmarshal(raw, &t); err != nil {
		return
	}
	topic := protocol.SanitizeText(t.Topic)
	if len([]rune(topic)) > 140 {
		topic = string([]rune(topic)[:140])
	}
	if strings.TrimSpace(topic) == "" {
		c.Send(protocol.NewPacket(protocol.TypeErrorMsg, protocol.ErrorMsgPayload{
			Code:    "INVALID_TOPIC",
			Message: "Topic cannot be empty",
		}))
		return
	}

	h.mu.RLock()
	room, exists := h.rooms[c.CurrentRoom]
	h.mu.RUnlock()
	if !exists {
		return
	}
	room.SetTopic(topic)
	room.Broadcast(protocol.NewPacket(protocol.TypeTopicAck, protocol.TopicPayload{
		Room:  room.Name,
		Topic: topic,
		By:    c.Nickname,
	}))
	room.Broadcast(protocol.NewPacket(protocol.TypeSystemMsg, protocol.SystemMsgPayload{
		Room:    room.Name,
		Content: fmt.Sprintf("Topic for %s set by %s: %s", room.Name, c.Nickname, topic),
		Level:   "info",
	}))
	h.broadcastRoomList()
}

// handleTyping relays ephemeral typing indicators to room peers (never stored).
func (h *Hub) handleTyping(c *Client, payload any) {
	raw, _ := json.Marshal(payload)
	var t protocol.TypingPayload
	if err := json.Unmarshal(raw, &t); err != nil {
		return
	}
	h.mu.RLock()
	room, exists := h.rooms[c.CurrentRoom]
	h.mu.RUnlock()
	if !exists {
		return
	}
	pkt := protocol.NewPacket(protocol.TypeTyping, protocol.TypingPayload{
		Room: room.Name,
		User: c.Nickname,
	})
	room.mu.RLock()
	defer room.mu.RUnlock()
	for member := range room.Members {
		if member != c {
			member.Send(pkt)
		}
	}
}

// Stats returns premium node telemetry for /health and logs.
func (h *Hub) Stats() (clients int, rooms int, totalMembers int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients = len(h.clients)
	rooms = len(h.rooms)
	for _, r := range h.rooms {
		totalMembers += r.MemberCount()
	}
	return clients, rooms, totalMembers
}

// broadcastUserList sends active users list to all members of a room.
func (h *Hub) broadcastUserList(r *Room) {
	users := r.MemberNicknames()
	r.Broadcast(protocol.NewPacket(protocol.TypeUserList, protocol.UserListPayload{
		Room:  r.Name,
		Users: users,
	}))
}

// broadcastRoomList pushes updated room list to all connected clients.
func (h *Hub) broadcastRoomList() {
	roomStatuses := make([]protocol.RoomStatus, 0, len(h.rooms))
	for _, r := range h.rooms {
		roomStatuses = append(roomStatuses, r.Status())
	}
	pkt := protocol.NewPacket(protocol.TypeRoomList, roomStatuses)
	for client := range h.clients {
		client.Send(pkt)
	}
}
