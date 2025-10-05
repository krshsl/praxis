package services

import (
	"log/slog"

	"github.com/spf13/viper"
)

// Config holds application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	AI        AIConfig
	JWT       JWTConfig
	WebSocket WebSocketConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL          string
	Seed         bool
	LogLevel     string
	MaxIdleConns int
	MaxOpenConns int
}

type AIConfig struct {
	GeminiAPIKey  string
	ElevenLabsKey string
}

type JWTConfig struct {
	Secret string
}

type WebSocketConfig struct {
	AllowedOrigins string
}

// LoadConfig loads configuration from environment variables and config files
func LoadConfig() *Config {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("websocket.allowed_origins", "")
	viper.SetDefault("gemini.api_key", "")
	viper.SetDefault("elevenlabs.api_key", "")
	viper.SetDefault("jwt.secret", "")
	viper.SetDefault("database.url", "")
	viper.SetDefault("database.seed", "true")
	viper.SetDefault("database.log_level", "silent")
	viper.SetDefault("database.max_idle_conns", "10")
	viper.SetDefault("database.max_open_conns", "100")

	// Map environment variables to config keys
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("websocket.allowed_origins", "WEBSOCKET_ALLOWED_ORIGINS")
	viper.BindEnv("gemini.api_key", "GEMINI_API_KEY")
	viper.BindEnv("elevenlabs.api_key", "ELEVENLABS_API_KEY")
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("database.url", "DATABASE_URL")
	viper.BindEnv("database.seed", "DATABASE_SEED")
	viper.BindEnv("database.log_level", "DATABASE_LOG_LEVEL")
	viper.BindEnv("database.max_idle_conns", "DATABASE_MAX_IDLE_CONNS")
	viper.BindEnv("database.max_open_conns", "DATABASE_MAX_OPEN_CONNS")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Warn("Config file not found, using defaults and environment variables")
		} else {
			slog.Error("Error reading config file", "error", err)
		}
	}

	return &Config{
		Server: ServerConfig{
			Port: viper.GetString("server.port"),
		},
		Database: DatabaseConfig{
			URL:          viper.GetString("database.url"),
			Seed:         viper.GetBool("database.seed"),
			LogLevel:     viper.GetString("database.log_level"),
			MaxIdleConns: viper.GetInt("database.max_idle_conns"),
			MaxOpenConns: viper.GetInt("database.max_open_conns"),
		},
		AI: AIConfig{
			GeminiAPIKey:  viper.GetString("gemini.api_key"),
			ElevenLabsKey: viper.GetString("elevenlabs.api_key"),
		},
		JWT: JWTConfig{
			Secret: viper.GetString("jwt.secret"),
		},
		WebSocket: WebSocketConfig{
			AllowedOrigins: viper.GetString("websocket.allowed_origins"),
		},
	}
}
