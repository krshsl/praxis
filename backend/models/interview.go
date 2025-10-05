package models

import (
	"time"

	"gorm.io/gorm"
)

// InterviewTranscript stores the ordered, turn-by-turn text of the conversation
type InterviewTranscript struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID string         `gorm:"type:uuid;not null;index" json:"session_id"`
	TurnOrder int            `gorm:"not null" json:"turn_order"` // Order of the turn in the conversation
	Speaker   string         `gorm:"not null;check:speaker IN ('user', 'agent')" json:"speaker"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Timestamp time.Time      `gorm:"not null" json:"timestamp"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Session InterviewSession `gorm:"foreignKey:SessionID" json:"session"`
}

// InterviewSummary stores the final AI-generated narrative analysis
type InterviewSummary struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID       string         `gorm:"type:uuid;not null;uniqueIndex" json:"session_id"`
	Summary         string         `gorm:"type:text;not null" json:"summary"` // Narrative summary
	Strengths       string         `gorm:"type:text" json:"strengths,omitempty"`
	Weaknesses      string         `gorm:"type:text" json:"weaknesses,omitempty"`
	Recommendations string         `gorm:"type:text" json:"recommendations,omitempty"`
	OverallScore    float64        `gorm:"type:decimal(5,2)" json:"overall_score"` // 0.00 to 100.00
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Session InterviewSession `gorm:"foreignKey:SessionID" json:"session"`
}

// PerformanceScore is a key-value table to store scores for various metrics
// This allows for future expansion without schema changes
type PerformanceScore struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID string         `gorm:"type:uuid;not null;index" json:"session_id"`
	Metric    string         `gorm:"not null" json:"metric"`                  // e.g., "communication", "technical_knowledge", "problem_solving"
	Score     float64        `gorm:"type:decimal(5,2);not null" json:"score"` // 0.00 to 100.00
	MaxScore  float64        `gorm:"type:decimal(5,2);not null;default:100.00" json:"max_score"`
	Weight    float64        `gorm:"type:decimal(3,2);not null;default:1.00" json:"weight"` // Weight for calculating overall score
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Session InterviewSession `gorm:"foreignKey:SessionID" json:"session"`
}
