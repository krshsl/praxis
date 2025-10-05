package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krshsl/praxis/backend/models"
	"gorm.io/gorm"
)

type ConversationRepository struct {
	db *gorm.DB
}

type ConversationHistory []string

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// SaveMessage saves a message to the database using GORM
func (r *ConversationRepository) SaveMessage(ctx context.Context, message *models.Message) error {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		slog.Error("Failed to save message", "error", err, "message_id", message.ID)
		return fmt.Errorf("failed to save message: %w", err)
	}

	slog.Info("Message saved", "message_id", message.ID, "user_id", message.UserID)
	return nil
}

// GetConversationHistory retrieves conversation history using GORM
func (r *ConversationRepository) GetConversationHistory(ctx context.Context, userID, sessionID string, limit int) ([]models.Message, error) {
	var messages []models.Message

	query := r.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("created_at DESC").
		Limit(limit)

	if err := query.Find(&messages).Error; err != nil {
		slog.Error("Failed to get conversation history", "error", err, "user_id", userID, "session_id", sessionID)
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	slog.Info("Conversation history retrieved", "user_id", userID, "session_id", sessionID, "count", len(messages))
	return messages, nil
}

// GetUserStats returns conversation statistics for a user using GORM
func (r *ConversationRepository) GetUserStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	var stats models.UserStats

	// Get total messages count
	if err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("user_id = ?", userID).
		Count(&stats.TotalMessages).Error; err != nil {
		slog.Error("Failed to get total messages count", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get total messages count: %w", err)
	}

	// Get total sessions count
	if err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("user_id = ?", userID).
		Distinct("session_id").
		Count(&stats.TotalSessions).Error; err != nil {
		slog.Error("Failed to get total sessions count", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get total sessions count: %w", err)
	}

	// Get code messages count
	if err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("user_id = ? AND message_type = ?", userID, "code").
		Count(&stats.CodeMessages).Error; err != nil {
		slog.Error("Failed to get code messages count", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get code messages count: %w", err)
	}

	// Get last activity
	var lastMessage models.Message
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&lastMessage).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			slog.Error("Failed to get last activity", "error", err, "user_id", userID)
			return nil, fmt.Errorf("failed to get last activity: %w", err)
		}
		// No messages found, last activity is nil
	} else {
		stats.LastActivity = &lastMessage.CreatedAt
	}

	slog.Info("User stats retrieved", "user_id", userID, "total_messages", stats.TotalMessages)

	return map[string]interface{}{
		"total_messages": stats.TotalMessages,
		"total_sessions": stats.TotalSessions,
		"code_messages":  stats.CodeMessages,
		"last_activity":  stats.LastActivity,
	}, nil
}

// DeleteUserMessages deletes all messages for a user using GORM
func (r *ConversationRepository) DeleteUserMessages(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.Message{}).Error; err != nil {
		slog.Error("Failed to delete user messages", "error", err, "user_id", userID)
		return fmt.Errorf("failed to delete user messages: %w", err)
	}

	slog.Info("User messages deleted", "user_id", userID)
	return nil
}

// GetMessagesBySession retrieves all messages for a specific session
func (r *ConversationRepository) GetMessagesBySession(ctx context.Context, sessionID string) ([]models.Message, error) {
	var messages []models.Message

	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		slog.Error("Failed to get messages by session", "error", err, "session_id", sessionID)
		return nil, fmt.Errorf("failed to get messages by session: %w", err)
	}

	return messages, nil
}

// GetMessageByID retrieves a specific message by ID
func (r *ConversationRepository) GetMessageByID(ctx context.Context, messageID string) (*models.Message, error) {
	var message models.Message

	if err := r.db.WithContext(ctx).
		Where("id = ?", messageID).
		First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("message not found: %s", messageID)
		}
		slog.Error("Failed to get message by ID", "error", err, "message_id", messageID)
		return nil, fmt.Errorf("failed to get message by ID: %w", err)
	}

	return &message, nil
}

// UpdateMessage updates an existing message
func (r *ConversationRepository) UpdateMessage(ctx context.Context, message *models.Message) error {
	if err := r.db.WithContext(ctx).
		Save(message).Error; err != nil {
		slog.Error("Failed to update message", "error", err, "message_id", message.ID)
		return fmt.Errorf("failed to update message: %w", err)
	}

	slog.Info("Message updated successfully", "message_id", message.ID)
	return nil
}

// DeleteMessage deletes a specific message
func (r *ConversationRepository) DeleteMessage(ctx context.Context, messageID string) error {
	if err := r.db.WithContext(ctx).
		Where("id = ?", messageID).
		Delete(&models.Message{}).Error; err != nil {
		slog.Error("Failed to delete message", "error", err, "message_id", messageID)
		return fmt.Errorf("failed to delete message: %w", err)
	}

	slog.Info("Message deleted successfully", "message_id", messageID)
	return nil
}

// GetRecentMessages retrieves recent messages for a user across all sessions
func (r *ConversationRepository) GetRecentMessages(ctx context.Context, userID string, limit int) ([]models.Message, error) {
	var messages []models.Message

	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		slog.Error("Failed to get recent messages", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}

	return messages, nil
}

// GetMessagesByType retrieves messages filtered by type (text or code)
func (r *ConversationRepository) GetMessagesByType(ctx context.Context, userID, messageType string, limit int) ([]models.Message, error) {
	var messages []models.Message

	query := r.db.WithContext(ctx).
		Where("user_id = ? AND message_type = ?", userID, messageType).
		Order("created_at DESC").
		Limit(limit)

	if err := query.Find(&messages).Error; err != nil {
		slog.Error("Failed to get messages by type", "error", err, "user_id", userID, "message_type", messageType)
		return nil, fmt.Errorf("failed to get messages by type: %w", err)
	}

	return messages, nil
}
