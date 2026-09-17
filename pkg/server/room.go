package server

import (
	"sync"

	"github.com/skywave-chat/skywave/pkg/crypto"
	"github.com/skywave-chat/skywave/pkg/protocol"
)

// Room represents a public or private chat room.
type Room struct {
	Name         string
	Topic        string
	PasswordHash string // Empty if public, hashed if private
	IsPrivate    bool
	MaxHistory   int
	History      []protocol.ChatMsgPayload // Ephemeral in-memory ring buffer
	Members      map[*Client]bool
	mu           sync.RWMutex
}

// NewRoom creates a room with specified privacy and in-memory history capacity.
func NewRoom(name string, topic string, password string, maxHistory int) *Room {
	r := &Room{
		Name:       name,
		Topic:      topic,
		MaxHistory: maxHistory,
		History:    make([]protocol.ChatMsgPayload, 0, maxHistory),
		Members:    make(map[*Client]bool),
	}
	if password != "" {
		r.IsPrivate = true
		r.PasswordHash = crypto.HashSecret(password, name)
	}
	return r
}

// Authenticate verifies password for private room access.
func (r *Room) Authenticate(password string) bool {
	if !r.IsPrivate {
		return true
	}
	return r.PasswordHash == crypto.HashSecret(password, r.Name)
}

// SetTopic updates the room topic thread-safely.
func (r *Room) SetTopic(topic string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Topic = topic
}

// AddMember adds a client to the room.
func (r *Room) AddMember(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Members[c] = true
}

// RemoveMember removes a client from the room. Returns remaining member count.
func (r *Room) RemoveMember(c *Client) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Members, c)
	return len(r.Members)
}

// HasMember checks if a client is in the room.
func (r *Room) HasMember(c *Client) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Members[c]
}

// MemberCount returns the number of active members.
func (r *Room) MemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Members)
}

// MemberNicknames returns a list of all active nicknames in this room.
func (r *Room) MemberNicknames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nicks := make([]string, 0, len(r.Members))
	for client := range r.Members {
		nicks = append(nicks, client.Nickname)
	}
	return nicks
}

// AddMessage appends a message to the in-memory circular history buffer.
func (r *Room) AddMessage(msg protocol.ChatMsgPayload) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.MaxHistory <= 0 {
		return // Zero-retention mode
	}

	if len(r.History) >= r.MaxHistory {
		// Drop oldest message
		r.History = append(r.History[1:], msg)
	} else {
		r.History = append(r.History, msg)
	}
}

// GetHistory returns a copy of the recent message buffer.
func (r *Room) GetHistory() []protocol.ChatMsgPayload {
	r.mu.RLock()
	defer r.mu.RUnlock()

	historyCopy := make([]protocol.ChatMsgPayload, len(r.History))
	copy(historyCopy, r.History)
	return historyCopy
}

// Broadcast sends a packet to all room members.
func (r *Room) Broadcast(packet protocol.Packet) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for client := range r.Members {
		client.Send(packet)
	}
}

// Status returns the public room metadata.
func (r *Room) Status() protocol.RoomStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return protocol.RoomStatus{
		Name:        r.Name,
		IsPrivate:   r.IsPrivate,
		MemberCount: len(r.Members),
		Topic:       r.Topic,
	}
}
