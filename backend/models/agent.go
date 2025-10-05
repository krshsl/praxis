package models

import (
	"time"

	"gorm.io/gorm"
)

// Agent represents both public agents (user_id is NULL) and private user-created agents (user_id is NOT NULL)
type Agent struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      *string        `gorm:"type:uuid;index" json:"user_id,omitempty"` // NULL for public agents
	Name        string         `gorm:"not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Personality string         `gorm:"type:text;not null" json:"personality"` // The AI personality/behavior
	Industry    string         `gorm:"size:100" json:"industry,omitempty"`
	Level       string         `gorm:"size:50" json:"level,omitempty"` // junior, mid, senior, executive
	IsPublic    bool           `gorm:"default:false" json:"is_public"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User              *User              `gorm:"foreignKey:UserID" json:"user,omitempty"`
	InterviewSessions []InterviewSession `gorm:"foreignKey:AgentID" json:"interview_sessions,omitempty"`
}

// InterviewSession represents each interview attempt, linking a user and an agent
type InterviewSession struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    string         `gorm:"type:uuid;not null;index" json:"user_id"`
	AgentID   string         `gorm:"type:uuid;not null;index" json:"agent_id"`
	Status    string         `gorm:"not null;default:'active';check:status IN ('active', 'completed', 'abandoned')" json:"status"`
	StartedAt time.Time      `gorm:"not null" json:"started_at"`
	EndedAt   *time.Time     `json:"ended_at,omitempty"`
	Duration  int            `json:"duration"` // Duration in seconds
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User              User                  `gorm:"foreignKey:UserID" json:"user"`
	Agent             Agent                 `gorm:"foreignKey:AgentID" json:"agent"`
	Transcripts       []InterviewTranscript `gorm:"foreignKey:SessionID" json:"transcripts,omitempty"`
	Summary           *InterviewSummary     `gorm:"foreignKey:SessionID" json:"summary,omitempty"`
	PerformanceScores []PerformanceScore    `gorm:"foreignKey:SessionID" json:"performance_scores,omitempty"`
}
