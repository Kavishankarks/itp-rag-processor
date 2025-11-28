package pipeline

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kavishankarks/document-hub/go-api/internal/embedding_client"
	"github.com/kavishankarks/document-hub/go-api/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Pipeline stages
const (
	StageParse     = "parse"
	StageSearch    = "search"
	StageNormalize = "normalize"
	StageChunk     = "chunk"
	StageEmbed     = "embed"
	StageStore     = "store"
)

// Pipeline statuses
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Orchestrator manages the RAG pipeline execution
type Orchestrator struct {
	DB              *gorm.DB // Exported for use in handlers
	embeddingClient *embedding_client.EmbeddingClient
	parser          *CurriculumParser
}

// NewOrchestrator creates a new pipeline orchestrator
func NewOrchestrator(db *gorm.DB, embeddingClient *embedding_client.EmbeddingClient) *Orchestrator {
	return &Orchestrator{
		DB:              db,
		embeddingClient: embeddingClient,
		parser:          NewCurriculumParser(),
	}
}

// StartPipeline initiates a new pipeline run
func (o *Orchestrator) StartPipeline(
	curriculum *models.Curriculum,
	config models.PipelineConfig,
) (*models.PipelineRun, error) {
	// Set default config values
	if config.ChunkSize == 0 {
		config.ChunkSize = 500
	}
	if config.ChunkOverlap == 0 {
		config.ChunkOverlap = 50
	}
	if config.SearchResultsPerTopic == 0 {
		config.SearchResultsPerTopic = 5
	}
	if config.SearchEngine == "" {
		config.SearchEngine = "duckduckgo"
	}

	// Marshal curriculum and config to JSON
	inputData, err := json.Marshal(curriculum)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal curriculum: %w", err)
	}

	configData, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create pipeline run record
	pipelineRun := &models.PipelineRun{
		CurriculumTitle: curriculum.Title,
		Status:          StatusPending,
		CurrentStage:    StageParse,
		InputData:       datatypes.JSON(inputData),
		Config:          datatypes.JSON(configData),
		Progress:        0,
	}

	if err := o.DB.Create(pipelineRun).Error; err != nil {
		return nil, fmt.Errorf("failed to create pipeline run: %w", err)
	}

	// Start processing asynchronously
	go o.processPipeline(pipelineRun.ID, curriculum, config)

	return pipelineRun, nil
}

// processPipeline executes the pipeline stages
func (o *Orchestrator) processPipeline(
	pipelineRunID uint,
	curriculum *models.Curriculum,
	config models.PipelineConfig,
) {
	// Update status to processing
	o.updatePipelineStatus(pipelineRunID, StatusProcessing, StageParse, 5, "")

	// Stage 1: Parse curriculum (already done, just extract topics)
	topics := o.parser.ExtractAllTopics(curriculum)
	log.Printf("Pipeline %d: Extracted %d topics", pipelineRunID, len(topics))

	// Create curriculum topic records
	if err := o.createTopicRecords(pipelineRunID, curriculum, topics); err != nil {
		o.updatePipelineStatus(pipelineRunID, StatusFailed, StageParse, 0, err.Error())
		return
	}

	o.updatePipelineStatus(pipelineRunID, StatusProcessing, StageSearch, 15, "")

	// Stage 2: Web Search (if enabled)
	if config.WebSearchEnabled {
		if err := o.enrichTopicsWithSearch(pipelineRunID, topics, config.SearchResultsPerTopic); err != nil {
			o.updatePipelineStatus(pipelineRunID, StatusFailed, StageSearch, 0, err.Error())
			return
		}
	} else {
		log.Printf("Pipeline %d: Web search disabled, using original content", pipelineRunID)
	}

	o.updatePipelineStatus(pipelineRunID, StatusProcessing, StageNormalize, 40, "")

	// Stage 3: Normalize content
	if err := o.normalizeTopics(pipelineRunID, config.Normalize); err != nil {
		o.updatePipelineStatus(pipelineRunID, StatusFailed, StageNormalize, 0, err.Error())
		return
	}

	o.updatePipelineStatus(pipelineRunID, StatusProcessing, StageChunk, 55, "")

	// Stage 4: Chunk and embed
	if err := o.chunkAndEmbedTopics(pipelineRunID, config); err != nil {
		o.updatePipelineStatus(pipelineRunID, StatusFailed, StageChunk, 0, err.Error())
		return
	}

	o.updatePipelineStatus(pipelineRunID, StatusProcessing, StageStore, 85, "")

	// Stage 5: Store documents (done in chunkAndEmbedTopics)
	log.Printf("Pipeline %d: All topics processed successfully", pipelineRunID)

	// Mark as completed
	o.updatePipelineStatus(pipelineRunID, StatusCompleted, StageStore, 100, "")
}

// createTopicRecords creates database records for each topic
func (o *Orchestrator) createTopicRecords(
	pipelineRunID uint,
	curriculum *models.Curriculum,
	topics []string,
) error {
	for _, topic := range topics {
		originalContent := o.parser.GenerateTopicContext(curriculum, topic)

		curriculumTopic := &models.CurriculumTopic{
			PipelineRunID:   pipelineRunID,
			TopicName:       topic,
			OriginalContent: originalContent,
			Status:          StatusPending,
		}

		if err := o.DB.Create(curriculumTopic).Error; err != nil {
			return fmt.Errorf("failed to create topic record for %s: %w", topic, err)
		}
	}

	return nil
}

// enrichTopicsWithSearch performs web search for each topic
func (o *Orchestrator) enrichTopicsWithSearch(
	pipelineRunID uint,
	topics []string,
	maxResults int,
) error {
	for i, topic := range topics {
		log.Printf("Pipeline %d: Searching for topic %d/%d: %s", pipelineRunID, i+1, len(topics), topic)

		// Call embedding service to enrich topic
		enrichedData, err := o.embeddingClient.EnrichTopic(topic, maxResults)
		if err != nil {
			log.Printf("Warning: Failed to enrich topic %s: %v", topic, err)
			// Continue with next topic instead of failing the entire pipeline
			continue
		}

		// Update topic with enriched content
		searchResultsJSON, _ := json.Marshal(enrichedData["results"])

		err = o.DB.Model(&models.CurriculumTopic{}).
			Where("pipeline_run_id = ? AND topic_name = ?", pipelineRunID, topic).
			Updates(map[string]interface{}{
				"enriched_content": enrichedData["combined_content"],
				"search_results":   datatypes.JSON(searchResultsJSON),
				"status":           "searching",
			}).Error

		if err != nil {
			return fmt.Errorf("failed to update topic %s: %w", topic, err)
		}

		// Small delay to avoid rate limiting
		time.Sleep(1 * time.Second)
	}

	return nil
}

// normalizeTopics normalizes the content for each topic
func (o *Orchestrator) normalizeTopics(pipelineRunID uint, shouldNormalize bool) error {
	if !shouldNormalize {
		log.Printf("Pipeline %d: Normalization disabled", pipelineRunID)
		return nil
	}

	var topics []models.CurriculumTopic
	if err := o.DB.Where("pipeline_run_id = ?", pipelineRunID).Find(&topics).Error; err != nil {
		return fmt.Errorf("failed to fetch topics: %w", err)
	}

	for i, topic := range topics {
		log.Printf("Pipeline %d: Normalizing topic %d/%d: %s", pipelineRunID, i+1, len(topics), topic.TopicName)

		// Get the content to normalize (enriched if available, otherwise original)
		content := topic.EnrichedContent
		if content == "" {
			content = topic.OriginalContent
		}

		// Call embedding service to normalize
		normalizedText, err := o.embeddingClient.NormalizeText(content, true)
		if err != nil {
			log.Printf("Warning: Failed to normalize topic %s: %v", topic.TopicName, err)
			// Use original content if normalization fails
			normalizedText = content
		}

		// Update the enriched content with normalized version
		err = o.DB.Model(&models.CurriculumTopic{}).
			Where("id = ?", topic.ID).
			Update("enriched_content", normalizedText).Error

		if err != nil {
			return fmt.Errorf("failed to update normalized content for %s: %w", topic.TopicName, err)
		}
	}

	return nil
}

// chunkAndEmbedTopics chunks, embeds, and stores documents for each topic
func (o *Orchestrator) chunkAndEmbedTopics(pipelineRunID uint, config models.PipelineConfig) error {
	var topics []models.CurriculumTopic
	if err := o.DB.Where("pipeline_run_id = ?", pipelineRunID).Find(&topics).Error; err != nil {
		return fmt.Errorf("failed to fetch topics: %w", err)
	}

	for i, topic := range topics {
		log.Printf("Pipeline %d: Processing topic %d/%d: %s", pipelineRunID, i+1, len(topics), topic.TopicName)

		// Get final content (enriched if available, otherwise original)
		content := topic.EnrichedContent
		if content == "" {
			content = topic.OriginalContent
		}

		// Create document for this topic
		document := &models.Document{
			Title:   topic.TopicName,
			Content: content,
			DocType: "curriculum_topic",
			Metadata: datatypes.JSON([]byte(fmt.Sprintf(
				`{"pipeline_run_id": %d, "source": "pipeline"}`,
				pipelineRunID,
			))),
		}

		// Chunk the content
		chunks, err := o.embeddingClient.ChunkText(content, config.ChunkSize)
		if err != nil {
			return fmt.Errorf("failed to chunk content for %s: %w", topic.TopicName, err)
		}

		// Generate embeddings for all chunks
		embeddings, err := o.embeddingClient.GetEmbeddings(chunks)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings for %s: %w", topic.TopicName, err)
		}

		// Store document and chunks in transaction
		err = o.DB.Transaction(func(tx *gorm.DB) error {
			// Create document
			if err := tx.Create(document).Error; err != nil {
				return fmt.Errorf("failed to create document: %w", err)
			}

			// Create chunks with embeddings
			for j, chunk := range chunks {
				documentChunk := &models.DocumentChunk{
					DocumentID: document.ID,
					ChunkText:  chunk,
					ChunkIndex: j,
					Embedding:  pgvector.NewVector(embeddings[j]),
				}

				if err := tx.Create(documentChunk).Error; err != nil {
					return fmt.Errorf("failed to create chunk: %w", err)
				}
			}

			// Update topic with document ID
			docID := document.ID
			return tx.Model(&models.CurriculumTopic{}).
				Where("id = ?", topic.ID).
				Updates(map[string]interface{}{
					"document_id": &docID,
					"status":      StatusCompleted,
				}).Error
		})

		if err != nil {
			return fmt.Errorf("failed to store document and chunks for %s: %w", topic.TopicName, err)
		}

		// Update progress
		progress := 85 + int(float64(i+1)/float64(len(topics))*10)
		o.updatePipelineStatus(pipelineRunID, StatusProcessing, StageStore, progress, "")
	}

	return nil
}

// updatePipelineStatus updates the pipeline run status
func (o *Orchestrator) updatePipelineStatus(
	pipelineRunID uint,
	status string,
	stage string,
	progress int,
	errorMessage string,
) {
	updates := map[string]interface{}{
		"status":        status,
		"current_stage": stage,
		"progress":      progress,
		"updated_at":    time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if err := o.DB.Model(&models.PipelineRun{}).Where("id = ?", pipelineRunID).Updates(updates).Error; err != nil {
		log.Printf("Error updating pipeline status: %v", err)
	}
}

// GetPipelineStatus retrieves the current status of a pipeline run
func (o *Orchestrator) GetPipelineStatus(pipelineRunID uint) (*models.PipelineStatusResponse, error) {
	var pipelineRun models.PipelineRun
	if err := o.DB.First(&pipelineRun, pipelineRunID).Error; err != nil {
		return nil, fmt.Errorf("pipeline run not found: %w", err)
	}

	// Build stages map
	stages := o.buildStagesMap(&pipelineRun)

	return &models.PipelineStatusResponse{
		ID:           pipelineRun.ID,
		Status:       pipelineRun.Status,
		CurrentStage: pipelineRun.CurrentStage,
		Progress:     pipelineRun.Progress,
		Stages:       stages,
		ErrorMessage: pipelineRun.ErrorMessage,
		CreatedAt:    pipelineRun.CreatedAt,
		UpdatedAt:    pipelineRun.UpdatedAt,
	}, nil
}

// buildStagesMap builds a map of stage statuses
func (o *Orchestrator) buildStagesMap(pipelineRun *models.PipelineRun) map[string]string {
	allStages := []string{StageParse, StageSearch, StageNormalize, StageChunk, StageEmbed, StageStore}
	stages := make(map[string]string)

	currentStageIndex := -1
	for i, stage := range allStages {
		if stage == pipelineRun.CurrentStage {
			currentStageIndex = i
			break
		}
	}

	for i, stage := range allStages {
		if pipelineRun.Status == StatusCompleted {
			stages[stage] = StatusCompleted
		} else if pipelineRun.Status == StatusFailed {
			if i < currentStageIndex {
				stages[stage] = StatusCompleted
			} else if i == currentStageIndex {
				stages[stage] = StatusFailed
			} else {
				stages[stage] = StatusPending
			}
		} else {
			if i < currentStageIndex {
				stages[stage] = StatusCompleted
			} else if i == currentStageIndex {
				stages[stage] = "in_progress"
			} else {
				stages[stage] = StatusPending
			}
		}
	}

	return stages
}

// GetPipelineResults retrieves the results of a completed pipeline run
func (o *Orchestrator) GetPipelineResults(pipelineRunID uint) (*models.PipelineResultsResponse, error) {
	var pipelineRun models.PipelineRun
	if err := o.DB.Preload("Topics").First(&pipelineRun, pipelineRunID).Error; err != nil {
		return nil, fmt.Errorf("pipeline run not found: %w", err)
	}

	// Get all documents created by this pipeline
	var documents []models.Document
	if err := o.DB.Where("metadata->>'pipeline_run_id' = ?", fmt.Sprintf("%d", pipelineRunID)).
		Preload("Chunks").
		Find(&documents).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}

	// Count total chunks
	totalChunks := 0
	for _, doc := range documents {
		totalChunks += len(doc.Chunks)
	}

	return &models.PipelineResultsResponse{
		PipelineRun: pipelineRun,
		Documents:   documents,
		TotalChunks: totalChunks,
	}, nil
}

// CancelPipeline cancels a running pipeline
func (o *Orchestrator) CancelPipeline(pipelineRunID uint) error {
	var pipelineRun models.PipelineRun
	if err := o.DB.First(&pipelineRun, pipelineRunID).Error; err != nil {
		return fmt.Errorf("pipeline run not found: %w", err)
	}

	if pipelineRun.Status == StatusCompleted || pipelineRun.Status == StatusFailed {
		return fmt.Errorf("cannot cancel pipeline in %s status", pipelineRun.Status)
	}

	// Update status to failed with cancellation message
	return o.DB.Model(&models.PipelineRun{}).
		Where("id = ?", pipelineRunID).
		Updates(map[string]interface{}{
			"status":        StatusFailed,
			"error_message": "Pipeline cancelled by user",
			"updated_at":    time.Now(),
		}).Error
}
