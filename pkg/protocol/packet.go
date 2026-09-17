package protocol

import "time"

// PacketType defines the type of message sent across the wire.
type PacketType string

const (
	// Client -> Server
	TypeAuth      PacketType = "auth"
	TypeJoinRoom  PacketType = "join_room"
	TypeLeaveRoom PacketType = "leave_room"
	TypeChatMsg   PacketType = "chat_msg"
	TypeDirectMsg PacketType = "direct_msg"
	TypeSetNick   PacketType = "set_nick"
	TypeSetTopic  PacketType = "set_topic"
	TypeTyping    PacketType = "typing"
	TypePing      PacketType = "ping"

	// Server -> Client
	TypeAuthAck   PacketType = "auth_ack"
	TypeNickAck   PacketType = "nick_ack"
	TypeTopicAck  PacketType = "topic_ack"
	TypeRoomList  PacketType = "room_list"
	TypeUserList  PacketType = "user_list"
	TypeSystemMsg PacketType = "system_msg"
	TypeErrorMsg  PacketType = "error_msg"
	TypePong      PacketType = "pong"
)

// BasePacket wraps all incoming and outgoing messages.
type Packet struct {
	Type      PacketType `json:"type"`
	Timestamp int64      `json:"timestamp,omitempty"`
	Payload   any        `json:"payload"`
}

// AuthPayload is sent by client on initial connection.
type AuthPayload struct {
	Nickname       string `json:"nickname"`
	AccountID      string `json:"account_id,omitempty"`
	ServerPassword string `json:"server_password,omitempty"`
	ClientVersion  string `json:"client_version"`
}

// AuthAckPayload is sent by server when authentication succeeds.
type AuthAckPayload struct {
	SessionID  string       `json:"session_id"`
	Nickname   string       `json:"nickname"`
	ServerName string       `json:"server_name"`
	MOTD       string       `json:"motd"`
	Rooms      []RoomStatus `json:"rooms"`
}

// RoomStatus provides metadata about an active room.
type RoomStatus struct {
	Name        string `json:"name"`
	IsPrivate   bool   `json:"is_private"`
	MemberCount int    `json:"member_count"`
	Topic       string `json:"topic,omitempty"`
}

// JoinRoomPayload is sent by client to join/create a room.
type JoinRoomPayload struct {
	RoomName string `json:"room_name"`
	Password string `json:"password,omitempty"` // Hashed or shared secret for private rooms
}

// ChatMsgPayload represents a public or room-level chat message.
type ChatMsgPayload struct {
	ID          string `json:"id"`
	Room        string `json:"room"`
	Sender      string `json:"sender"`
	Content     string `json:"content"`
	IsEncrypted bool   `json:"is_encrypted,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// DirectMsgPayload represents a 1-on-1 private message.
type DirectMsgPayload struct {
	ID          string `json:"id"`
	Sender      string `json:"sender"`
	Recipient   string `json:"recipient"`
	Content     string `json:"content"`
	IsEncrypted bool   `json:"is_encrypted,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// SetNickPayload requests a nickname change.
type SetNickPayload struct {
	NewNickname string `json:"new_nickname"`
}

// NickAckPayload confirms a nickname change.
type NickAckPayload struct {
	OldNickname string `json:"old_nickname"`
	NewNickname string `json:"new_nickname"`
}

// SystemMsgPayload sends announcement or room notification.
type SystemMsgPayload struct {
	Room    string `json:"room,omitempty"`
	Content string `json:"content"`
	Level   string `json:"level"` // "info", "warn", "success"
}

// ErrorMsgPayload reports errors back to the client.
type ErrorMsgPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UserListPayload reports active users in a room or server.
type UserListPayload struct {
	Room  string   `json:"room"`
	Users []string `json:"users"`
}

// TopicPayload sets or announces a room topic.
type TopicPayload struct {
	Room  string `json:"room"`
	Topic string `json:"topic"`
	By    string `json:"by,omitempty"`
}

// TypingPayload is an ephemeral "user is typing" signal (never stored).
type TypingPayload struct {
	Room string `json:"room"`
	User string `json:"user"`
}

// NewPacket creates a packet with the current unix millisecond timestamp.
func NewPacket(pType PacketType, payload any) Packet {
	return Packet{
		Type:      pType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}
