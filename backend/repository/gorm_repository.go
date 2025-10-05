package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/krshsl/praxis/backend/models"
	"gorm.io/gorm"
)

type GORMRepository struct {
	db *gorm.DB
}

func NewGORMRepository(db *gorm.DB) *GORMRepository {
	return &GORMRepository{db: db}
}

// AutoMigrate runs database migrations
func (r *GORMRepository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&models.User{},
		&models.Agent{},
		&models.InterviewSession{},
		&models.InterviewTranscript{},
		&models.InterviewSummary{},
		&models.PerformanceScore{},
		&models.RefreshToken{},
		&models.PermanentToken{},
		&models.Message{},
	)
}

// User operations
func (r *GORMRepository) CreateUser(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		slog.Error("Failed to create user", "error", err)
		return err
	}
	slog.Info("User created", "user_id", user.ID, "email", user.Email)
	return nil
}

func (r *GORMRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get user by email", "error", err, "email", email)
		return nil, err
	}
	return &user, nil
}

func (r *GORMRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get user by ID", "error", err, "user_id", id)
		return nil, err
	}
	return &user, nil
}

// Note: Old Session and Message models have been replaced with InterviewSession and InterviewTranscript
// These operations are now handled by the interview-specific methods below

// Token operations
func (r *GORMRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		slog.Error("Failed to create refresh token", "error", err)
		return err
	}
	return nil
}

func (r *GORMRepository) GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	if err := r.db.WithContext(ctx).Where("token = ? AND expires_at > ?", token, time.Now()).First(&refreshToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get refresh token", "error", err)
		return nil, err
	}
	return &refreshToken, nil
}

func (r *GORMRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	if err := r.db.WithContext(ctx).Where("token = ?", token).Delete(&models.RefreshToken{}).Error; err != nil {
		slog.Error("Failed to delete refresh token", "error", err)
		return err
	}
	return nil
}

func (r *GORMRepository) CreatePermanentToken(ctx context.Context, token *models.PermanentToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		slog.Error("Failed to create permanent token", "error", err)
		return err
	}
	return nil
}

func (r *GORMRepository) GetPermanentToken(ctx context.Context, token string) (*models.PermanentToken, error) {
	var permanentToken models.PermanentToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&permanentToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get permanent token", "error", err)
		return nil, err
	}
	return &permanentToken, nil
}

func (r *GORMRepository) DeletePermanentToken(ctx context.Context, token string) error {
	if err := r.db.WithContext(ctx).Where("token = ?", token).Delete(&models.PermanentToken{}).Error; err != nil {
		slog.Error("Failed to delete permanent token", "error", err)
		return err
	}
	return nil
}

func (r *GORMRepository) DeleteAllUserTokens(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error; err != nil {
		slog.Error("Failed to delete user refresh tokens", "error", err, "user_id", userID)
		return err
	}
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.PermanentToken{}).Error; err != nil {
		slog.Error("Failed to delete user permanent tokens", "error", err, "user_id", userID)
		return err
	}
	return nil
}

// Interview-specific operations using GORM ORM
func (r *GORMRepository) CreateAgent(ctx context.Context, agent *models.Agent) error {
	if err := r.db.WithContext(ctx).Create(agent).Error; err != nil {
		slog.Error("Failed to create agent", "error", err)
		return err
	}
	slog.Info("Agent created", "agent_id", agent.ID, "name", agent.Name)
	return nil
}

func (r *GORMRepository) GetAgents(ctx context.Context, userID string, includePublic bool) ([]models.Agent, error) {
	var agents []models.Agent
	query := r.db.WithContext(ctx).Where("is_active = ?", true)

	if includePublic {
		if userID == "" {
			// When userID is empty, only get public agents (user_id IS NULL)
			query = query.Where("user_id IS NULL")
		} else {
			// When userID is provided, get both public agents and user's private agents
			query = query.Where("(user_id IS NULL OR user_id = ?)", userID)
		}
	} else {
		// Only get user's private agents
		if userID == "" {
			// If no userID provided, return empty result
			return agents, nil
		}
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&agents).Error; err != nil {
		slog.Error("Failed to get agents", "error", err, "user_id", userID)
		return nil, err
	}
	return agents, nil
}

func (r *GORMRepository) CreateInterviewSession(ctx context.Context, session *models.InterviewSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		slog.Error("Failed to create interview session", "error", err)
		return err
	}
	slog.Info("Interview session created", "session_id", session.ID, "user_id", session.UserID)
	return nil
}

func (r *GORMRepository) GetInterviewSessions(ctx context.Context, userID string) ([]models.InterviewSession, error) {
	var sessions []models.InterviewSession
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Preload("Agent").Find(&sessions).Error
	if err != nil {
		slog.Error("Failed to get interview sessions", "error", err, "user_id", userID)
		return nil, err
	}
	return sessions, nil
}

func (r *GORMRepository) CreateInterviewTranscript(ctx context.Context, transcript *models.InterviewTranscript) error {
	if err := r.db.WithContext(ctx).Create(transcript).Error; err != nil {
		slog.Error("Failed to create interview transcript", "error", err)
		return err
	}
	slog.Info("Interview transcript created", "transcript_id", transcript.ID, "session_id", transcript.SessionID)
	return nil
}

func (r *GORMRepository) GetInterviewTranscripts(ctx context.Context, sessionID string) ([]models.InterviewTranscript, error) {
	var transcripts []models.InterviewTranscript
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("turn_order").Find(&transcripts).Error
	if err != nil {
		slog.Error("Failed to get interview transcripts", "error", err, "session_id", sessionID)
		return nil, err
	}
	return transcripts, nil
}

func (r *GORMRepository) CreateInterviewSummary(ctx context.Context, summary *models.InterviewSummary) error {
	if err := r.db.WithContext(ctx).Create(summary).Error; err != nil {
		slog.Error("Failed to create interview summary", "error", err)
		return err
	}
	slog.Info("Interview summary created", "summary_id", summary.ID, "session_id", summary.SessionID)
	return nil
}

func (r *GORMRepository) GetInterviewSummary(ctx context.Context, sessionID string) (*models.InterviewSummary, error) {
	var summary models.InterviewSummary
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&summary).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get interview summary", "error", err, "session_id", sessionID)
		return nil, err
	}
	return &summary, nil
}

func (r *GORMRepository) CreatePerformanceScore(ctx context.Context, score *models.PerformanceScore) error {
	if err := r.db.WithContext(ctx).Create(score).Error; err != nil {
		slog.Error("Failed to create performance score", "error", err)
		return err
	}
	slog.Info("Performance score created", "score_id", score.ID, "session_id", score.SessionID, "metric", score.Metric)
	return nil
}

func (r *GORMRepository) GetPerformanceScores(ctx context.Context, sessionID string) ([]models.PerformanceScore, error) {
	var scores []models.PerformanceScore
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Find(&scores).Error
	if err != nil {
		slog.Error("Failed to get performance scores", "error", err, "session_id", sessionID)
		return nil, err
	}
	return scores, nil
}

// Additional methods needed by endpoints

func (r *GORMRepository) GetAgentByID(ctx context.Context, agentID string, userID string) (*models.Agent, error) {
	var agent models.Agent
	// Get agent if it's public OR belongs to the user
	err := r.db.WithContext(ctx).Where("id = ? AND (user_id IS NULL OR user_id = ?)", agentID, userID).First(&agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get agent by ID", "error", err, "agent_id", agentID, "user_id", userID)
		return nil, err
	}
	return &agent, nil
}

func (r *GORMRepository) UpdateAgent(ctx context.Context, agent *models.Agent) error {
	if err := r.db.WithContext(ctx).Save(agent).Error; err != nil {
		slog.Error("Failed to update agent", "error", err, "agent_id", agent.ID)
		return err
	}
	slog.Info("Agent updated", "agent_id", agent.ID, "name", agent.Name)
	return nil
}

func (r *GORMRepository) DeleteAgent(ctx context.Context, agentID string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", agentID).Delete(&models.Agent{}).Error; err != nil {
		slog.Error("Failed to delete agent", "error", err, "agent_id", agentID)
		return err
	}
	slog.Info("Agent deleted", "agent_id", agentID)
	return nil
}

func (r *GORMRepository) GetInterviewSessionWithDetails(ctx context.Context, sessionID string, userID string) (*models.InterviewSession, error) {
	var session models.InterviewSession
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Preload("Agent").
		Preload("Transcripts").
		Preload("Summary").
		Preload("PerformanceScores").
		First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get interview session with details", "error", err, "session_id", sessionID, "user_id", userID)
		return nil, err
	}
	return &session, nil
}

// GetInterviewSession gets an interview session by ID without user check
func (r *GORMRepository) GetInterviewSession(ctx context.Context, sessionID string) (*models.InterviewSession, error) {
	var session models.InterviewSession
	err := r.db.WithContext(ctx).
		Where("id = ?", sessionID).
		First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get interview session", "error", err, "session_id", sessionID)
		return nil, err
	}
	return &session, nil
}

// GetAgent gets an agent by ID
func (r *GORMRepository) GetAgent(ctx context.Context, agentID string) (*models.Agent, error) {
	var agent models.Agent
	err := r.db.WithContext(ctx).
		Where("id = ?", agentID).
		First(&agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		slog.Error("Failed to get agent", "error", err, "agent_id", agentID)
		return nil, err
	}
	return &agent, nil
}
