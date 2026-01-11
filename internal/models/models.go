package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:120;not null" json:"name"`
	Email        string    `gorm:"size:190;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"size:20;not null" json:"type"` // "direct"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Participants []ConversationParticipant `json:"-"`
}

type ConversationParticipant struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"index;not null" json:"conversation_id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"index;not null" json:"conversation_id"`
	SenderID       uint      `gorm:"index;not null" json:"sender_id"`
	Body           string    `gorm:"type:text;not null" json:"body"`
	CreatedAt      time.Time `gorm:"index" json:"created_at"`
}
