package main

import (
	"log/slog"
	"os"

	"github.com/krshsl/praxis/backend/repository"
	"github.com/krshsl/praxis/backend/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var (
	gormDB   *gorm.DB
	gormRepo *repository.GORMRepository
)

func main() {
	// Setup structured logging with JSON format
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	config := services.LoadConfig()

	// Initialize database connection
	var err error
	if config.Database.URL != "" {
		// Configure GORM logger based on config
		var gormLogLevel gormLogger.LogLevel
		switch config.Database.LogLevel {
		case "silent":
			gormLogLevel = gormLogger.Silent
		case "error":
			gormLogLevel = gormLogger.Error
		case "warn":
			gormLogLevel = gormLogger.Warn
		case "info":
			gormLogLevel = gormLogger.Info
		default:
			gormLogLevel = gormLogger.Silent
		}

		// Initialize GORM for ORM operations with PostgreSQL
		gormDB, err = gorm.Open(postgres.Open(config.Database.URL), &gorm.Config{
			// Disable foreign key constraint checks during migration for better performance
			DisableForeignKeyConstraintWhenMigrating: true,
			// Skip default transaction for better performance
			SkipDefaultTransaction: true,
			// Configure logging level
			Logger: gormLogger.Default.LogMode(gormLogLevel),
		})
		if err != nil {
			slog.Error("Failed to connect to database with GORM", "error", err)
		} else {
			slog.Info("Connected to database with GORM")

			// Configure database connection pool for better performance
			if sqlDB, err := gormDB.DB(); err == nil {
				// Set connection pool settings from config
				sqlDB.SetMaxIdleConns(config.Database.MaxIdleConns) // Maximum number of idle connections
				sqlDB.SetMaxOpenConns(config.Database.MaxOpenConns) // Maximum number of open connections
				sqlDB.SetConnMaxLifetime(0)                         // Connection lifetime (0 = unlimited)
				slog.Info("Database connection pool configured",
					"max_idle_conns", config.Database.MaxIdleConns,
					"max_open_conns", config.Database.MaxOpenConns)
			}

			// Initialize GORM repository
			gormRepo = repository.NewGORMRepository(gormDB)

			// Auto-migrate database tables
			if err := gormRepo.AutoMigrate(); err != nil {
				slog.Error("Failed to auto-migrate database tables", "error", err)
			} else {
				slog.Info("Database tables migrated successfully")
			}

			// Seed database with initial data (if enabled)
			if config.Database.Seed {
				seeder := services.NewDatabaseSeeder(gormRepo)
				if err := seeder.SeedDatabase(); err != nil {
					slog.Error("Failed to seed database", "error", err)
				} else {
					slog.Info("Database seeded successfully")
				}
			} else {
				slog.Info("Database seeding disabled")
			}
		}
	} else {
		slog.Warn("Database URL not configured, running without database")
	}

	// Initialize server
	server := services.NewServer(config)
	server.SetDatabase(gormRepo, gormDB)

	// Initialize all services
	if err := server.InitializeServices(); err != nil {
		slog.Error("Failed to initialize services", "error", err)
		os.Exit(1)
	}

	// Start the server
	server.Start()
}
