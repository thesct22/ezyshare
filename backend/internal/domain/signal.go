package domain

// MessageType defines custom string types for signal protocol actions
type MessageType string

const (
	TypeJoin      MessageType = "join"
	TypeLeave     MessageType = "leave"
	TypeOffer     MessageType = "offer"
	TypeAnswer    MessageType = "answer"
	TypeCandidate MessageType = "candidate"
)

// SignalMessage is the struct representing JSON payloads exchanged over WebSockets.
type SignalMessage struct {
	Type     MessageType `json:"type"`
	SenderID string      `json:"sender_id"`
	TargetID string      `json:"target_id,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

// Client is an interface that abstracts a connected peer connection.
type Client interface {
	ID() string
	Send(msg SignalMessage) error
	Close() error
}
