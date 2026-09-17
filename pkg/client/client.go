package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/skywave-chat/skywave/pkg/crypto"
	"github.com/skywave-chat/skywave/pkg/protocol"
)

// Client handles WebSocket connection to a Skywave server.
type Client struct {
	ServerURL      string
	Nickname       string
	AccountID      string
	ServerPassword string
	CurrentRoom    string
	SessionID      string
	ServerName     string
	MOTD           string

	conn        *websocket.Conn
	roomKeys    map[string]string // roomName -> decryption passphrase
	mu          sync.RWMutex
	isConnected bool

	// OnNickChanged is invoked (from the read pump goroutine) when the
	// server confirms a nickname change. UI uses it to persist identity.
	OnNickChanged func(oldNick, newNick string)

	// Channels for UI integration
	IncomingPackets chan protocol.Packet
	Errors          chan error
	DisconnectChan  chan struct{}
}

// NewClient initializes a new Skywave client instance.
func NewClient(serverURL, nickname, serverPassword string) *Client {
	return NewClientWithAccount(serverURL, nickname, "", serverPassword)
}

// NewClientWithAccount initializes a client bound to a local anonymous account.
func NewClientWithAccount(serverURL, nickname, accountID, serverPassword string) *Client {
	return &Client{
		ServerURL:       serverURL,
		Nickname:        nickname,
		AccountID:       accountID,
		ServerPassword:  serverPassword,
		CurrentRoom:     "#general",
		roomKeys:        make(map[string]string),
		IncomingPackets: make(chan protocol.Packet, 256),
		Errors:          make(chan error, 32),
		DisconnectChan:  make(chan struct{}),
	}
}

// Connect establishes WebSocket connection and starts reader pump.
func (c *Client) Connect() error {
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", u.String(), err)
	}

	c.mu.Lock()
	c.conn = conn
	c.isConnected = true
	c.mu.Unlock()

	// Launch read loop
	go c.readPump()

	// Send initial Auth packet only if nickname is already set
	if c.Nickname != "" {
		return c.Authenticate(c.Nickname)
	}
	return nil
}

// Authenticate sets the nickname and sends the Auth packet to the server.
func (c *Client) Authenticate(nickname string) error {
	c.mu.Lock()
	c.Nickname = nickname
	c.mu.Unlock()

	authPacket := protocol.NewPacket(protocol.TypeAuth, protocol.AuthPayload{
		Nickname:       nickname,
		AccountID:      c.AccountID,
		ServerPassword: c.ServerPassword,
		ClientVersion:  "2.0.0",
	})
	return c.SendPacket(authPacket)
}

// SetRoomKey stores a local secret passphrase for a private room for End-to-End Encryption.
func (c *Client) SetRoomKey(room string, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roomKeys[room] = key
}

// GetRoomKey retrieves the local secret passphrase for a private room.
func (c *Client) GetRoomKey(room string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roomKeys[room]
}

// SendChat sends a chat message, encrypting if room has a secret key.
func (c *Client) SendChat(text string) error {
	c.mu.RLock()
	room := c.CurrentRoom
	key := c.roomKeys[room]
	c.mu.RUnlock()

	content := text
	isEncrypted := false

	if key != "" {
		cipher, err := crypto.Encrypt(text, key)
		if err == nil {
			content = cipher
			isEncrypted = true
		}
	}

	pkt := protocol.NewPacket(protocol.TypeChatMsg, protocol.ChatMsgPayload{
		Room:        room,
		Content:     content,
		IsEncrypted: isEncrypted,
	})
	return c.SendPacket(pkt)
}

// SendDirectMsg sends a private direct message to a user.
func (c *Client) SendDirectMsg(recipient, text string) error {
	pkt := protocol.NewPacket(protocol.TypeDirectMsg, protocol.DirectMsgPayload{
		Recipient: recipient,
		Content:   text,
	})
	return c.SendPacket(pkt)
}

// JoinRoom sends request to switch/create room.
func (c *Client) JoinRoom(roomName, password string) error {
	c.mu.Lock()
	c.CurrentRoom = roomName
	if password != "" {
		c.roomKeys[roomName] = password
	}
	c.mu.Unlock()

	pkt := protocol.NewPacket(protocol.TypeJoinRoom, protocol.JoinRoomPayload{
		RoomName: roomName,
		Password: password,
	})
	return c.SendPacket(pkt)
}

// ChangeNick sends request to update nickname.
func (c *Client) ChangeNick(newNick string) error {
	clean, err := protocol.ValidateNickname(newNick)
	if err != nil {
		return err
	}
	pkt := protocol.NewPacket(protocol.TypeSetNick, protocol.SetNickPayload{
		NewNickname: clean,
	})
	return c.SendPacket(pkt)
}

// SetTopic sets the topic of the current room.
func (c *Client) SetTopic(topic string) error {
	c.mu.RLock()
	room := c.CurrentRoom
	c.mu.RUnlock()
	return c.SendPacket(protocol.NewPacket(protocol.TypeSetTopic, protocol.TopicPayload{
		Room:  room,
		Topic: topic,
	}))
}

// SendTyping emits an ephemeral typing indicator (caller should throttle).
func (c *Client) SendTyping() error {
	c.mu.RLock()
	room := c.CurrentRoom
	nick := c.Nickname
	c.mu.RUnlock()
	return c.SendPacket(protocol.NewPacket(protocol.TypeTyping, protocol.TypingPayload{
		Room: room,
		User: nick,
	}))
}

// SendPacket writes a packet as JSON to the server.
func (c *Client) SendPacket(p protocol.Packet) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	return c.conn.WriteJSON(p)
}

// Disconnect gracefully disconnects from server.
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected && c.conn != nil {
		_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "leaving"))
		c.conn.Close()
		c.isConnected = false
	}
}

// IsConnected returns whether client is currently connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// SwitchServer disconnects from current server and seamlessly dials a new server.
func (c *Client) SwitchServer(newURL, password string) error {
	c.Disconnect()

	c.mu.Lock()
	c.ServerURL = newURL
	c.ServerPassword = password
	c.DisconnectChan = make(chan struct{})
	c.CurrentRoom = "#general"
	c.mu.Unlock()

	return c.Connect()
}

// readPump receives packets from server and processes encryption/payloads.
func (c *Client) readPump() {
	defer func() {
		c.mu.Lock()
		c.isConnected = false
		c.mu.Unlock()
		close(c.DisconnectChan)
	}()

	for {
		var p protocol.Packet
		err := c.conn.ReadJSON(&p)
		if err != nil {
			c.Errors <- err
			break
		}

		// Process specific packets to update client state
		switch p.Type {
		case protocol.TypeAuthAck:
			raw, _ := json.Marshal(p.Payload)
			var ack protocol.AuthAckPayload
			if err := json.Unmarshal(raw, &ack); err == nil {
				c.mu.Lock()
				c.SessionID = ack.SessionID
				c.Nickname = ack.Nickname
				c.ServerName = ack.ServerName
				c.MOTD = ack.MOTD
				c.mu.Unlock()
			}

		case protocol.TypeChatMsg:
			raw, _ := json.Marshal(p.Payload)
			var chat protocol.ChatMsgPayload
			if err := json.Unmarshal(raw, &chat); err == nil {
				// If message is encrypted and we have key, decrypt it locally
				if chat.IsEncrypted {
					key := c.GetRoomKey(chat.Room)
					if key != "" {
						plain, err := crypto.Decrypt(chat.Content, key)
						if err == nil {
							chat.Content = plain
						} else {
							chat.Content = "[◈ Encrypted message — unable to decrypt]"
						}
					} else {
						chat.Content = "[◈ Encrypted message — key required]"
					}
					p.Payload = chat
				}
			}

		case protocol.TypeNickAck:
			raw, _ := json.Marshal(p.Payload)
			var ack protocol.NickAckPayload
			if err := json.Unmarshal(raw, &ack); err == nil && ack.NewNickname != "" {
				var cb func(string, string)
				c.mu.Lock()
				old := c.Nickname
				c.Nickname = ack.NewNickname
				cb = c.OnNickChanged
				c.mu.Unlock()
				if cb != nil {
					cb(old, ack.NewNickname)
				}
			}
		}

		c.IncomingPackets <- p
	}
}
