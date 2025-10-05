package models

// This file serves as the central export point for all database models
// Import this package to access all model types

// All models are automatically exported from their respective files:
// - User, RefreshToken, PermanentToken from user.go
// - Agent, InterviewSession from agent.go
// - InterviewTranscript, InterviewSummary, PerformanceScore from interview.go
// - Message, UserStats from message.go

// Database schema overview:
// 1. users - Managed by cookie-based authentication
// 2. agents - Both public agents (user_id is NULL) and private user-created agents
// 3. interview_sessions - Records each interview attempt, linking a user and an agent
// 4. interview_transcripts - Stores the ordered, turn-by-turn text of the conversation
// 5. interview_summaries - Stores the final AI-generated narrative analysis
// 6. performance_scores - A key-value table to store scores for various metrics
