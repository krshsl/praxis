package models

import (
	"time"

	"gorm.io/gorm"
)

// Message represents a single message in a conversation
type Message struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(255);default:gen_random_uuid()"`
	UserID      string         `json:"user_id" gorm:"type:uuid;not null;index"`
	SessionID   string         `json:"session_id" gorm:"type:varchar(255);not null;index"`
	Content     string         `json:"content" gorm:"type:text;not null"`
	Role        string         `json:"role" gorm:"type:varchar(50);not null;check:role IN ('user', 'assistant')"`
	MessageType string         `json:"message_type" gorm:"type:varchar(50);not null;check:message_type IN ('text', 'code')"`
	Language    *string        `json:"language,omitempty" gorm:"type:varchar(50)"`
	CreatedAt   time.Time      `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User    User              `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Session *InterviewSession `json:"session,omitempty" gorm:"foreignKey:SessionID;references:ID"`
}

// TableName returns the table name for the Message model
func (Message) TableName() string {
	return "messages"
}

// BeforeCreate hook to set the ID if not provided
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		// GORM will handle UUID generation via default:gen_random_uuid()
		// This hook is kept for any additional logic if needed
	}
	return nil
}

// UserStats represents aggregated statistics for a user
type UserStats struct {
	TotalMessages int64      `json:"total_messages"`
	TotalSessions int64      `json:"total_sessions"`
	CodeMessages  int64      `json:"code_messages"`
	LastActivity  *time.Time `json:"last_activity"`
}
