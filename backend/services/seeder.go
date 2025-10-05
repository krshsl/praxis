package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krshsl/praxis/backend/models"
	"github.com/krshsl/praxis/backend/repository"
	"golang.org/x/crypto/bcrypt"
)

// DatabaseSeeder handles database seeding operations
type DatabaseSeeder struct {
	repo *repository.GORMRepository
}

// NewDatabaseSeeder creates a new database seeder
func NewDatabaseSeeder(repo *repository.GORMRepository) *DatabaseSeeder {
	return &DatabaseSeeder{repo: repo}
}

// SeedDatabase seeds the database with initial data (idempotent)
func (s *DatabaseSeeder) SeedDatabase() error {
	ctx := context.Background()

	// Check if seeding has already been completed
	if s.isSeedingComplete(ctx) {
		slog.Info("Database seeding already completed, skipping")
		return nil
	}

	// Hash default password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create test users (no admin users for security)
	users := []models.User{
		{
			Email:     "test@example.com",
			Password:  string(hashedPassword),
			FullName:  "Test User",
			AvatarURL: "",
			Role:      "user",
		},
		{
			Email:     "demo@example.com",
			Password:  string(hashedPassword),
			FullName:  "Demo User",
			AvatarURL: "",
			Role:      "user",
		},
	}

	// Seed users (idempotent)
	for _, user := range users {
		if err := s.seedUser(ctx, user); err != nil {
			slog.Error("Failed to seed user", "email", user.Email, "error", err)
		}
	}

	// Get the first user for creating private agents
	firstUser, err := s.repo.GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		return fmt.Errorf("failed to get test user: %w", err)
	}
	if firstUser == nil {
		return fmt.Errorf("test user not found")
	}

	// Create default agents (always public)
	defaultAgents := []models.Agent{
		{
			UserID:      nil, // Public agent
			Name:        "Sarah Chen - Tech Recruiter",
			Description: "Experienced technical recruiter specializing in software engineering roles",
			Personality: "Professional, encouraging, and detail-oriented. Asks thoughtful technical questions and provides constructive feedback.",
			Industry:    "Technology",
			Level:       "Senior",
			IsPublic:    true,
			IsActive:    true,
		},
		{
			UserID:      nil, // Public agent
			Name:        "Marcus Johnson - Product Manager",
			Description: "Senior product manager with expertise in product strategy and team leadership",
			Personality: "Strategic thinker who focuses on product vision, user experience, and cross-functional collaboration.",
			Industry:    "Product Management",
			Level:       "Senior",
			IsPublic:    true,
			IsActive:    true,
		},
		{
			UserID:      nil, // Public agent
			Name:        "Dr. Emily Rodriguez - Data Scientist",
			Description: "Lead data scientist with expertise in machine learning and statistical analysis",
			Personality: "Analytical and methodical, focuses on problem-solving approach and technical depth in data science.",
			Industry:    "Data Science",
			Level:       "Senior",
			IsPublic:    true,
			IsActive:    true,
		},
		{
			UserID:      nil, // Public agent
			Name:        "Alex Thompson - Frontend Developer",
			Description: "Senior frontend developer with expertise in React, Vue, and modern web technologies",
			Personality: "Creative and technically focused, emphasizes clean code, user experience, and modern development practices.",
			Industry:    "Frontend Development",
			Level:       "Senior",
			IsPublic:    true,
			IsActive:    true,
		},
		{
			UserID:      nil, // Public agent
			Name:        "Lisa Wang - Backend Engineer",
			Description: "Senior backend engineer specializing in distributed systems and cloud architecture",
			Personality: "Systematic and performance-oriented, focuses on scalability, security, and system design principles.",
			Industry:    "Backend Development",
			Level:       "Senior",
			IsPublic:    true,
			IsActive:    true,
		},
		{
			UserID:      nil, // Public agent
			Name:        "David Kim - DevOps Engineer",
			Description: "DevOps engineer with expertise in CI/CD, containerization, and cloud infrastructure",
			Personality: "Process-oriented and automation-focused, emphasizes reliability, monitoring, and infrastructure as code.",
			Industry:    "DevOps",
			Level:       "Senior",
			IsPublic:    true,
			IsActive:    true,
		},
	}

	// Seed default agents (idempotent)
	for _, agent := range defaultAgents {
		if err := s.seedAgent(ctx, agent); err != nil {
			slog.Error("Failed to seed agent", "name", agent.Name, "error", err)
		}
	}

	// Create private agent for test user
	privateAgent := models.Agent{
		UserID:      &firstUser.ID, // Private agent
		Name:        "My Custom Interviewer",
		Description: "A personalized interviewer for my specific needs",
		Personality: "Adaptive and supportive, tailored to my learning style and career goals.",
		Industry:    "General",
		Level:       "Mid",
		IsPublic:    false,
		IsActive:    true,
	}

	// Seed private agent for test user (idempotent)
	if err := s.seedAgent(ctx, privateAgent); err != nil {
		slog.Error("Failed to seed private agent", "error", err)
	}

	// Mark seeding as complete
	if err := s.markSeedingComplete(ctx); err != nil {
		slog.Error("Failed to mark seeding as complete", "error", err)
	}

	return nil
}

// isSeedingComplete checks if seeding has already been completed
func (s *DatabaseSeeder) isSeedingComplete(ctx context.Context) bool {
	// Check if we have the expected number of default agents
	agents, err := s.repo.GetAgents(ctx, "", true) // Get all public agents
	if err != nil {
		return false
	}

	// Count public agents (should be 6 default agents)
	publicAgentCount := 0
	for _, agent := range agents {
		if agent.UserID == nil && agent.IsPublic {
			publicAgentCount++
		}
	}

	// If we have all 6 default agents, seeding is likely complete
	return publicAgentCount >= 6
}

// markSeedingComplete marks seeding as complete (could be implemented with a seeding table)
func (s *DatabaseSeeder) markSeedingComplete(ctx context.Context) error {
	// For now, we rely on the presence of default agents to determine completion
	// In a more robust implementation, you could create a seeding_metadata table
	slog.Info("Database seeding completed successfully")
	return nil
}

// seedUser seeds a single user (idempotent)
func (s *DatabaseSeeder) seedUser(ctx context.Context, user models.User) error {
	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("error checking user %s: %w", user.Email, err)
	}

	if existingUser != nil {
		slog.Info("User already exists, skipping", "email", user.Email)
		return nil
	}

	// User doesn't exist, create it
	if err := s.repo.CreateUser(ctx, &user); err != nil {
		return fmt.Errorf("failed to create user %s: %w", user.Email, err)
	}

	slog.Info("Created user", "email", user.Email)
	return nil
}

// seedAgent seeds a single agent (idempotent)
func (s *DatabaseSeeder) seedAgent(ctx context.Context, agent models.Agent) error {
	// For public agents, check by name and public status
	if agent.UserID == nil {
		agents, err := s.repo.GetAgents(ctx, "", true) // Get all public agents
		if err != nil {
			return fmt.Errorf("error checking agents: %w", err)
		}

		for _, existingAgent := range agents {
			if existingAgent.Name == agent.Name && existingAgent.UserID == nil {
				slog.Info("Public agent already exists, skipping", "name", agent.Name)
				return nil
			}
		}
	} else {
		// For private agents, check by name and user ID
		agents, err := s.repo.GetAgents(ctx, *agent.UserID, false) // Get user's private agents
		if err != nil {
			return fmt.Errorf("error checking private agents: %w", err)
		}

		for _, existingAgent := range agents {
			if existingAgent.Name == agent.Name {
				slog.Info("Private agent already exists, skipping", "name", agent.Name, "user_id", *agent.UserID)
				return nil
			}
		}
	}

	// Agent doesn't exist, create it
	if err := s.repo.CreateAgent(ctx, &agent); err != nil {
		return fmt.Errorf("failed to create agent %s: %w", agent.Name, err)
	}

	slog.Info("Created agent", "name", agent.Name, "is_public", agent.UserID == nil)
	return nil
}
