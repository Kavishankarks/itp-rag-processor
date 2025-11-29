package models

import (
	"time"
)

// Document represents a stored documentation
type Document struct {
	ID        uint                   `json:"id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	SourceURL string                 `json:"source_url,omitempty"`
	DocType   string                 `json:"doc_type,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Chunks    []DocumentChunk        `json:"chunks,omitempty"`
}

// DocumentChunk represents a chunk of a document with its embedding
type DocumentChunk struct {
	ID         uint      `json:"id"`
	DocumentID uint      `json:"document_id"`
	ChunkText  string    `json:"chunk_text"`
	ChunkIndex int       `json:"chunk_index"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateDocumentRequest represents the request to create a document
type CreateDocumentRequest struct {
	Title     string                 `json:"title" validate:"required" example:"PostgreSQL Best Practices"`
	Content   string                 `json:"content" validate:"required" example:"PostgreSQL is a powerful database..."`
	SourceURL string                 `json:"source_url,omitempty" example:"https://example.com/docs"`
	DocType   string                 `json:"doc_type,omitempty" example:"tutorial"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" swaggertype:"object"`
}

// UpdateDocumentRequest represents the request to update a document
type UpdateDocumentRequest struct {
	Title     *string                `json:"title,omitempty"`
	Content   *string                `json:"content,omitempty"`
	SourceURL *string                `json:"source_url,omitempty"`
	DocType   *string                `json:"doc_type,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" swaggertype:"object"`
}

// SearchRequest represents a search query
type SearchRequest struct {
	Query      string  `json:"query" validate:"required"`
	SearchType string  `json:"search_type,omitempty"` // fulltext, semantic, hybrid
	Limit      int     `json:"limit,omitempty"`
}

// SearchResult represents a search result with score
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
	Snippet  string   `json:"snippet,omitempty"`
}

// PipelineRun represents a pipeline execution
type PipelineRun struct {
	ID              uint                   `json:"id"`
	CurriculumTitle string                 `json:"curriculum_title"`
	Status          string                 `json:"status"` // pending, processing, completed, failed
	CurrentStage    string                 `json:"current_stage,omitempty"`  // parse, search, normalize, chunk, embed, store
	InputData       map[string]interface{} `json:"input_data" swaggertype:"object"`
	Config          map[string]interface{} `json:"config" swaggertype:"object"`
	Progress        int                    `json:"progress"` // 0-100
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Topics          []CurriculumTopic      `json:"topics,omitempty"`
}

// CurriculumTopic represents a topic within a curriculum
type CurriculumTopic struct {
	ID              uint                   `json:"id"`
	PipelineRunID   uint                   `json:"pipeline_run_id"`
	TopicName       string                 `json:"topic_name"`
	OriginalContent string                 `json:"original_content,omitempty"`
	EnrichedContent string                 `json:"enriched_content,omitempty"`
	SearchResults   map[string]interface{} `json:"search_results,omitempty" swaggertype:"object"`
	Status          string                 `json:"status"` // pending, searching, processing, completed, failed
	DocumentID      *uint                  `json:"document_id,omitempty"` // Reference to created document
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// PipelineConfig represents pipeline configuration
type PipelineConfig struct {
	WebSearchEnabled      bool   `json:"web_search_enabled"`
	SearchResultsPerTopic int    `json:"search_results_per_topic"`
	ChunkSize             int    `json:"chunk_size"`
	ChunkOverlap          int    `json:"chunk_overlap"`
	Normalize             bool   `json:"normalize"`
	SearchEngine          string `json:"search_engine"` // duckduckgo, brave
}

// Curriculum represents a course curriculum structure
type Curriculum struct {
	Title   string             `json:"title" validate:"required"`
	Modules []CurriculumModule `json:"modules" validate:"required"`
}

// CurriculumModule represents a module in a curriculum
type CurriculumModule struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description,omitempty"`
	Topics      []string `json:"topics" validate:"required"`
}

// StartPipelineRequest represents the request to start a pipeline
type StartPipelineRequest struct {
	Curriculum Curriculum     `json:"curriculum" validate:"required"`
	Config     PipelineConfig `json:"config"`
}

// PipelineStatusResponse represents the response for pipeline status
type PipelineStatusResponse struct {
	ID           uint                  `json:"id"`
	Status       string                `json:"status"`
	CurrentStage string                `json:"current_stage"`
	Progress     int                   `json:"progress"`
	Stages       map[string]string     `json:"stages"`
	ErrorMessage string                `json:"error_message,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// PipelineResultsResponse represents the response for pipeline results
type PipelineResultsResponse struct {
	PipelineRun PipelineRun `json:"pipeline_run"`
	Documents   []Document  `json:"documents"`
	TotalChunks int         `json:"total_chunks"`
}
