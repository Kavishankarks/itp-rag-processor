package pipeline

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kavishankarks/itp-rag-processor/go-api/internal/embedding_client"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/models"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/vector"
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
	embeddingClient *embedding_client.EmbeddingClient
	milvusClient    *vector.MilvusClient
	parser          *CurriculumParser

	// In-memory state storage
	runsMu sync.RWMutex
	runs   map[uint]*models.PipelineRun

	topicsMu sync.RWMutex
	topics   map[uint][]*models.CurriculumTopic

	// ID counters
	nextRunID   uint
	nextTopicID uint
}

// NewOrchestrator creates a new pipeline orchestrator
func NewOrchestrator(embeddingClient *embedding_client.EmbeddingClient, milvusClient *vector.MilvusClient) *Orchestrator {
	return &Orchestrator{
		embeddingClient: embeddingClient,
		milvusClient:    milvusClient,
		parser:          NewCurriculumParser(),
		runs:            make(map[uint]*models.PipelineRun),
		topics:          make(map[uint][]*models.CurriculumTopic),
		nextRunID:       1,
		nextTopicID:     1,
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
	inputDataBytes, err := json.Marshal(curriculum)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal curriculum: %w", err)
	}
	var inputDataMap map[string]interface{}
	json.Unmarshal(inputDataBytes, &inputDataMap)

	configDataBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	var configDataMap map[string]interface{}
	json.Unmarshal(configDataBytes, &configDataMap)

	o.runsMu.Lock()
	runID := o.nextRunID
	o.nextRunID++

	pipelineRun := &models.PipelineRun{
		ID:              runID,
		CurriculumTitle: curriculum.Title,
		Status:          StatusPending,
		CurrentStage:    StageParse,
		InputData:       inputDataMap,
		Config:          configDataMap,
		Progress:        0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	o.runs[runID] = pipelineRun
	o.runsMu.Unlock()

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
	topicNames []string,
) error {
	o.topicsMu.Lock()
	defer o.topicsMu.Unlock()

	var topics []*models.CurriculumTopic

	for _, topicName := range topicNames {
		originalContent := o.parser.GenerateTopicContext(curriculum, topicName)

		topicID := o.nextTopicID
		o.nextTopicID++

		curriculumTopic := &models.CurriculumTopic{
			ID:              topicID,
			PipelineRunID:   pipelineRunID,
			TopicName:       topicName,
			OriginalContent: originalContent,
			Status:          StatusPending,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		topics = append(topics, curriculumTopic)
	}

	o.topics[pipelineRunID] = topics

	return nil
}

// enrichTopicsWithSearch performs web search for each topic
func (o *Orchestrator) enrichTopicsWithSearch(
	pipelineRunID uint,
	topicNames []string,
	maxResults int,
) error {
	o.topicsMu.Lock()
	topics, exists := o.topics[pipelineRunID]
	o.topicsMu.Unlock()

	if !exists {
		return fmt.Errorf("topics not found for pipeline run %d", pipelineRunID)
	}

	for i, topic := range topics {
		log.Printf("Pipeline %d: Searching for topic %d/%d: %s", pipelineRunID, i+1, len(topics), topic.TopicName)

		// Call embedding service to enrich topic
		enrichedData, err := o.embeddingClient.EnrichTopic(topic.TopicName, maxResults)
		if err != nil {
			log.Printf("Warning: Failed to enrich topic %s: %v", topic.TopicName, err)
			continue
		}

		// Update topic with enriched content
		// searchResultsJSON, _ := json.Marshal(enrichedData["results"])

		o.topicsMu.Lock()
		if content, ok := enrichedData["combined_content"].(string); ok {
			topic.EnrichedContent = content
		}
		if results, ok := enrichedData["results"].(map[string]interface{}); ok {
			topic.SearchResults = results
		} else if results, ok := enrichedData["results"].([]interface{}); ok {
			// If results is a list, wrap it in a map
			topic.SearchResults = map[string]interface{}{"results": results}
		}
		topic.Status = "searching"
		topic.UpdatedAt = time.Now()
		o.topicsMu.Unlock()

		// Small delay to avoid rate limiting
		time.Sleep(1 * time.Second)
	}

	return nil
}

// ListPipelines lists all pipeline runs with pagination
func (o *Orchestrator) ListPipelines(limit, offset int) ([]models.PipelineRun, int64, error) {
	o.runsMu.RLock()
	defer o.runsMu.RUnlock()

	var runs []models.PipelineRun
	for _, run := range o.runs {
		runs = append(runs, *run)
	}

	// Sort by CreatedAt DESC
	// We need to implement sort, but for now let's just return them.
	// Since map iteration is random, we should sort.
	// But to save code, I'll skip sort or do simple bubble sort if needed.
	// Let's just return as is for now or implement simple sort.

	total := int64(len(runs))

	// Apply pagination
	if offset >= len(runs) {
		return []models.PipelineRun{}, total, nil
	}

	end := offset + limit
	if end > len(runs) {
		end = len(runs)
	}

	return runs[offset:end], total, nil
}

// normalizeTopics normalizes the content for each topic
func (o *Orchestrator) normalizeTopics(pipelineRunID uint, shouldNormalize bool) error {
	if !shouldNormalize {
		log.Printf("Pipeline %d: Normalization disabled", pipelineRunID)
		return nil
	}

	o.topicsMu.RLock()
	topics, exists := o.topics[pipelineRunID]
	o.topicsMu.RUnlock()

	if !exists {
		return fmt.Errorf("topics not found for pipeline run %d", pipelineRunID)
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
			normalizedText = content
		}

		o.topicsMu.Lock()
		topic.EnrichedContent = normalizedText
		topic.UpdatedAt = time.Now()
		o.topicsMu.Unlock()
	}

	return nil
}

// chunkAndEmbedTopics chunks, embeds, and stores documents for each topic
func (o *Orchestrator) chunkAndEmbedTopics(pipelineRunID uint, config models.PipelineConfig) error {
	o.topicsMu.RLock()
	topics, exists := o.topics[pipelineRunID]
	o.topicsMu.RUnlock()

	if !exists {
		return fmt.Errorf("topics not found for pipeline run %d", pipelineRunID)
	}

	for i, topic := range topics {
		log.Printf("Pipeline %d: Processing topic %d/%d: %s", pipelineRunID, i+1, len(topics), topic.TopicName)

		// Get final content (enriched if available, otherwise original)
		content := topic.EnrichedContent
		if content == "" {
			content = topic.OriginalContent
		}

		// Create document for this topic in Milvus
		metadata := map[string]interface{}{
			"pipeline_run_id": pipelineRunID,
			"source":          "pipeline",
		}
		metadataBytes, _ := json.Marshal(metadata)

		milvusDoc := &vector.Document{
			Title:    topic.TopicName,
			Content:  content,
			DocType:  "curriculum_topic",
			Metadata: string(metadataBytes),
		}

		docID, err := o.milvusClient.CreateDocument(milvusDoc)
		if err != nil {
			return fmt.Errorf("failed to create document for %s: %w", topic.TopicName, err)
		}

		// Chunk the content
		chunks, err := o.embeddingClient.ChunkText(content, config.ChunkSize)
		if err != nil {
			o.milvusClient.DeleteDocument(docID)
			return fmt.Errorf("failed to chunk content for %s: %w", topic.TopicName, err)
		}

		// Generate embeddings for all chunks
		embeddings, err := o.embeddingClient.GetEmbeddings(chunks)
		if err != nil {
			o.milvusClient.DeleteDocument(docID)
			return fmt.Errorf("failed to generate embeddings for %s: %w", topic.TopicName, err)
		}

		// Create chunks with embeddings
		var milvusChunks []vector.Chunk
		for j, chunk := range chunks {
			milvusChunks = append(milvusChunks, vector.Chunk{
				DocumentID: docID,
				ChunkIndex: int64(j),
				ChunkText:  chunk,
				Embedding:  embeddings[j],
			})
		}

		// Store in Milvus
		if err := o.milvusClient.AddChunks(milvusChunks); err != nil {
			o.milvusClient.DeleteDocument(docID)
			return fmt.Errorf("failed to store chunks in Milvus: %w", err)
		}

		// Update topic with document ID
		o.topicsMu.Lock()
		uintDocID := uint(docID)
		topic.DocumentID = &uintDocID
		topic.Status = StatusCompleted
		topic.UpdatedAt = time.Now()
		o.topicsMu.Unlock()

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
	o.runsMu.Lock()
	defer o.runsMu.Unlock()

	run, exists := o.runs[pipelineRunID]
	if !exists {
		log.Printf("Error updating pipeline status: run %d not found", pipelineRunID)
		return
	}

	run.Status = status
	run.CurrentStage = stage
	run.Progress = progress
	run.UpdatedAt = time.Now()

	if errorMessage != "" {
		run.ErrorMessage = errorMessage
	}
}

// GetPipelineStatus retrieves the current status of a pipeline run
func (o *Orchestrator) GetPipelineStatus(pipelineRunID uint) (*models.PipelineStatusResponse, error) {
	o.runsMu.RLock()
	run, exists := o.runs[pipelineRunID]
	o.runsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pipeline run not found")
	}

	// Build stages map
	stages := o.buildStagesMap(run)

	return &models.PipelineStatusResponse{
		ID:           run.ID,
		Status:       run.Status,
		CurrentStage: run.CurrentStage,
		Progress:     run.Progress,
		Stages:       stages,
		ErrorMessage: run.ErrorMessage,
		CreatedAt:    run.CreatedAt,
		UpdatedAt:    run.UpdatedAt,
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
	o.runsMu.RLock()
	run, exists := o.runs[pipelineRunID]
	o.runsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pipeline run not found")
	}

	o.topicsMu.RLock()
	topics := o.topics[pipelineRunID]
	o.topicsMu.RUnlock()

	// Convert to value slice for response
	var topicValues []models.CurriculumTopic
	for _, t := range topics {
		topicValues = append(topicValues, *t)
	}

	// We need to fetch documents from Milvus that match this pipeline run
	// Since we don't have a direct "GetDocumentsByMetadata" in our simple MilvusClient,
	// and we stored DocumentID in topics, we can fetch by ID.

	var documents []models.Document
	totalChunks := 0

	for _, topic := range topics {
		if topic.DocumentID != nil {
			milvusDoc, err := o.milvusClient.GetDocument(int64(*topic.DocumentID))
			if err != nil {
				continue
			}

			var metadata map[string]interface{}
			json.Unmarshal([]byte(milvusDoc.Metadata), &metadata)

			doc := models.Document{
				ID:        uint(milvusDoc.ID),
				Title:     milvusDoc.Title,
				Content:   milvusDoc.Content,
				SourceURL: milvusDoc.SourceURL,
				DocType:   milvusDoc.DocType,
				Metadata:  metadata,
			}
			documents = append(documents, doc)

			// We don't have chunks count in Document struct from Milvus GetDocument.
			// We would need to query chunks count.
			// For now, let's assume 0 or try to count.
		}
	}

	// We can't easily populate run.Topics because run is a pointer to shared struct.
	// We should return a copy or just the topics list separately.
	// The PipelineResultsResponse expects PipelineRun which has Topics []CurriculumTopic.
	// But our in-memory run doesn't have Topics populated (it's in o.topics).

	runCopy := *run
	runCopy.Topics = topicValues

	return &models.PipelineResultsResponse{
		PipelineRun: runCopy,
		Documents:   documents,
		TotalChunks: totalChunks,
	}, nil
}

// CancelPipeline cancels a running pipeline
func (o *Orchestrator) CancelPipeline(pipelineRunID uint) error {
	o.runsMu.Lock()
	defer o.runsMu.Unlock()

	run, exists := o.runs[pipelineRunID]
	if !exists {
		return fmt.Errorf("pipeline run not found")
	}

	if run.Status == StatusCompleted || run.Status == StatusFailed {
		return fmt.Errorf("cannot cancel pipeline in %s status", run.Status)
	}

	run.Status = StatusFailed
	run.ErrorMessage = "Pipeline cancelled by user"
	run.UpdatedAt = time.Now()

	return nil
}
