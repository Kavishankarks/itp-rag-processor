package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/kavishankarks/document-hub/go-api/internal/models"
)

var DB *gorm.DB

// Initialize sets up the database connection and runs migrations
func Initialize() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	// Configure GORM logger
	logLevel := logger.Silent
	if os.Getenv("DEBUG") == "true" {
		logLevel = logger.Info
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable extensions
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return nil, fmt.Errorf("failed to create vector extension: %w", err)
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return nil, fmt.Errorf("failed to create pg_trgm extension: %w", err)
	}

	// Run migrations
	if err := db.AutoMigrate(
		&models.Document{},
		&models.DocumentChunk{},
		&models.PipelineRun{},
		&models.CurriculumTopic{},
	); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create unique index on title to prevent duplicates
	db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_title_unique
		ON documents (title)
	`)

	// Create GIN index for full-text search on documents
	db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_documents_content_gin
		ON documents USING gin(to_tsvector('english', content))
	`)

	// Create GIN index for full-text search on chunks
	db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunks_text_gin
		ON document_chunks USING gin(to_tsvector('english', chunk_text))
	`)

	// Create HNSW index for vector similarity search
	db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunks_embedding_hnsw
		ON document_chunks USING hnsw(embedding vector_cosine_ops)
	`)

	// Create index on pipeline runs for efficient status queries
	db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status
		ON pipeline_runs (status, created_at DESC)
	`)

	// Create index on curriculum topics for pipeline queries
	db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_curriculum_topics_status
		ON curriculum_topics (pipeline_run_id, status)
	`)

	DB = db
	return db, nil
}
