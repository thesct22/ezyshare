package domain

// MessageType defines custom string types for signal protocol actions
type MessageType string

const (
	TypeJoin        MessageType = "join"
	TypeLeave       MessageType = "leave"
	TypeOffer       MessageType = "offer"
	TypeAnswer      MessageType = "answer"
	TypeCandidate   MessageType = "candidate"
	TypeCreateRoom  MessageType = "create_room"
	TypeJoinRoom    MessageType = "join_room"
	TypeLeaveRoom   MessageType = "leave_room"
	TypePeerJoined  MessageType = "peer_joined"
	TypePeerLeft    MessageType = "peer_left"
	TypeRoomCreated MessageType = "room_created"
	TypeError       MessageType = "error"
)

// SignalMessage is the struct representing JSON payloads exchanged over WebSockets.
type SignalMessage struct {
	Type     MessageType `json:"type"`
	SenderID string      `json:"sender_id"`
	TargetID string      `json:"target_id,omitempty"`
	RoomID   string      `json:"room_id,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

// Client is an interface that abstracts a connected peer connection.
type Client interface {
	ID() string
	Send(msg SignalMessage) error
	Close() error
}
