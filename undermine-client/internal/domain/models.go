package domain

import (
	"time"
)

// MessageStatus отражает статус доставки сообщения
type MessageStatus string

const (
	StatusSending   MessageStatus = "sending"
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
	StatusFailed    MessageStatus = "failed"
)

// User представляет пользователя для отображения в интерфейсе
type User struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	IsOnline bool      `json:"is_online"`
	LastSeen time.Time `json:"last_seen"`
}

// Message представляет единичное сообщение в диалоге
type Message struct {
	ID         string        `json:"id"`
	ChatID     string        `json:"chat_id"`
	SenderID   string        `json:"sender_id"`
	Content    string        `json:"content"`
	Status     MessageStatus `json:"status"`
	IsOutgoing bool          `json:"is_outgoing"`
	CreatedAt  time.Time     `json:"created_at"`
}

// Chat представляет элемент диалога в боковом списке
type Chat struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	PeerUser    User      `json:"peer_user"`
	LastMessage *Message  `json:"last_message,omitempty"`
	UnreadCount int       `json:"unread_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}
