package vector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	CollectionName = "document_chunks"
	Dim            = 384 // Embedding dimension
)

type MilvusClient struct {
	client client.Client
}

type Chunk struct {
	ID         int64
	DocumentID int64
	ChunkIndex int64
	ChunkText  string
	Embedding  []float32
}

func Initialize(url, token string) (*MilvusClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := client.NewClient(ctx, client.Config{
		Address: url,
		APIKey:  token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Milvus: %w", err)
	}

	return &MilvusClient{client: c}, nil
}

func (m *MilvusClient) Close() {
	if m.client != nil {
		m.client.Close()
	}
}

func (m *MilvusClient) AddChunks(chunks []Chunk) error {
	ctx := context.Background()

	documentIDs := make([]int64, len(chunks))
	chunkIndices := make([]int64, len(chunks))
	chunkTexts := make([]string, len(chunks))
	embeddings := make([][]float32, len(chunks))

	for i, chunk := range chunks {
		documentIDs[i] = chunk.DocumentID
		chunkIndices[i] = chunk.ChunkIndex
		chunkTexts[i] = chunk.ChunkText
		embeddings[i] = chunk.Embedding
	}

	documentIDCol := entity.NewColumnInt64("document_id", documentIDs)
	chunkIndexCol := entity.NewColumnInt64("chunk_index", chunkIndices)
	chunkTextCol := entity.NewColumnVarChar("chunk_text", chunkTexts)
	embeddingCol := entity.NewColumnFloatVector("embedding", Dim, embeddings)

	_, err := m.client.Insert(ctx, CollectionName, "", documentIDCol, chunkIndexCol, chunkTextCol, embeddingCol)
	if err != nil {
		return fmt.Errorf("failed to insert chunks: %w", err)
	}

	return nil
}

type SearchResult struct {
	DocumentID int64
	ChunkText  string
	Score      float32
}

func (m *MilvusClient) Search(queryVector []float32, limit int, minScore float64) ([]SearchResult, error) {
	ctx := context.Background()

	sp, _ := entity.NewIndexFlatSearchParam() // AutoIndex uses default search params usually, or we can use specific ones if we knew the index type. AutoIndex is safe.

	searchResult, err := m.client.Search(
		ctx,
		CollectionName,
		[]string{},
		"",
		[]string{"document_id", "chunk_text"},
		[]entity.Vector{entity.FloatVector(queryVector)},
		"embedding",
		entity.COSINE,
		limit,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	var results []SearchResult
	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
			score := result.Scores[i]
			if score < float32(minScore) {
				continue
			}

			docID, err := result.Fields.GetColumn("document_id").Get(i)
			if err != nil {
				log.Printf("Error getting document_id: %v", err)
				continue
			}

			chunkText, err := result.Fields.GetColumn("chunk_text").Get(i)
			if err != nil {
				log.Printf("Error getting chunk_text: %v", err)
				continue
			}

			results = append(results, SearchResult{
				DocumentID: docID.(int64),
				ChunkText:  chunkText.(string),
				Score:      score,
			})
		}
	}

	return results, nil
}

// Document represents a document in Milvus
type Document struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	SourceURL string `json:"source_url"`
	DocType   string `json:"doc_type"`
	Metadata  string `json:"metadata"` // JSON string
	CreatedAt int64  `json:"created_at"`
}

const DocumentsCollection = "documents"

func (m *MilvusClient) EnsureCollections() error {
	ctx := context.Background()

	// 1. Document Chunks Collection
	if err := m.ensureChunksCollection(ctx); err != nil {
		return err
	}

	// 2. Documents Collection
	if err := m.ensureDocumentsCollection(ctx); err != nil {
		return err
	}

	return nil
}

func (m *MilvusClient) ensureChunksCollection(ctx context.Context) error {
	has, err := m.client.HasCollection(ctx, CollectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !has {
		schema := &entity.Schema{
			CollectionName: CollectionName,
			Description:    "Document chunks for RAG",
			AutoID:         true,
			Fields: []*entity.Field{
				{
					Name:       "id",
					DataType:   entity.FieldTypeInt64,
					PrimaryKey: true,
					AutoID:     true,
				},
				{
					Name:     "document_id",
					DataType: entity.FieldTypeInt64,
				},
				{
					Name:     "chunk_index",
					DataType: entity.FieldTypeInt64,
				},
				{
					Name:     "chunk_text",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						entity.TypeParamMaxLength: "65535",
					},
				},
				{
					Name:     "embedding",
					DataType: entity.FieldTypeFloatVector,
					TypeParams: map[string]string{
						entity.TypeParamDim: fmt.Sprintf("%d", Dim),
					},
				},
			},
		}

		if err := m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}

		// Create index on embedding
		idx, err := entity.NewIndexAUTOINDEX(entity.COSINE)
		if err != nil {
			return fmt.Errorf("failed to create index definition: %w", err)
		}

		if err := m.client.CreateIndex(ctx, CollectionName, "embedding", idx, false); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}

		// Load collection
		if err := m.client.LoadCollection(ctx, CollectionName, false); err != nil {
			return fmt.Errorf("failed to load collection: %w", err)
		}
	}
	return nil
}

func (m *MilvusClient) ensureDocumentsCollection(ctx context.Context) error {
	has, err := m.client.HasCollection(ctx, DocumentsCollection)
	if err != nil {
		return fmt.Errorf("failed to check documents collection existence: %w", err)
	}

	if !has {
		schema := &entity.Schema{
			CollectionName: DocumentsCollection,
			Description:    "Documents metadata",
			AutoID:         true,
			Fields: []*entity.Field{
				{
					Name:       "id",
					DataType:   entity.FieldTypeInt64,
					PrimaryKey: true,
					AutoID:     true,
				},
				{
					Name:     "title",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						entity.TypeParamMaxLength: "1024",
					},
				},
				{
					Name:     "content",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						entity.TypeParamMaxLength: "65535", // Milvus limit
					},
				},
				{
					Name:     "source_url",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						entity.TypeParamMaxLength: "2048",
					},
				},
				{
					Name:     "doc_type",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						entity.TypeParamMaxLength: "64",
					},
				},
				{
					Name:     "metadata",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						entity.TypeParamMaxLength: "65535",
					},
				},
				{
					Name:     "created_at",
					DataType: entity.FieldTypeInt64,
				},
				{
					Name:     "dummy_vector",
					DataType: entity.FieldTypeFloatVector,
					TypeParams: map[string]string{
						entity.TypeParamDim: "4",
					},
				},
			},
		}

		if err := m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
			return fmt.Errorf("failed to create documents collection: %w", err)
		}

		// Create index on dummy_vector (required for loading)
		idx, err := entity.NewIndexAUTOINDEX(entity.L2)
		if err != nil {
			return fmt.Errorf("failed to create index definition for documents: %w", err)
		}

		if err := m.client.CreateIndex(ctx, DocumentsCollection, "dummy_vector", idx, false); err != nil {
			return fmt.Errorf("failed to create index for documents: %w", err)
		}

		if err := m.client.LoadCollection(ctx, DocumentsCollection, false); err != nil {
			return fmt.Errorf("failed to load documents collection: %w", err)
		}
	}
	return nil
}

// CreateDocument creates a new document in Milvus and returns its ID
func (m *MilvusClient) CreateDocument(doc *Document) (int64, error) {
	ctx := context.Background()

	// Check for duplicates by title
	// Query signature: ctx, collection, partitions, expr, outputFields
	existing, err := m.client.Query(ctx, DocumentsCollection, []string{}, fmt.Sprintf("title == \"%s\"", doc.Title), []string{"id"})
	if err == nil && existing.GetColumn("id").Len() > 0 {
		return 0, fmt.Errorf("duplicate key value: document with title '%s' already exists", doc.Title)
	}

	titleCol := entity.NewColumnVarChar("title", []string{doc.Title})
	contentCol := entity.NewColumnVarChar("content", []string{doc.Content})
	sourceURLCol := entity.NewColumnVarChar("source_url", []string{doc.SourceURL})
	docTypeCol := entity.NewColumnVarChar("doc_type", []string{doc.DocType})
	metadataCol := entity.NewColumnVarChar("metadata", []string{doc.Metadata})
	createdAtCol := entity.NewColumnInt64("created_at", []int64{time.Now().Unix()})

	// Dummy vector
	dummyVector := []float32{0.0, 0.0, 0.0, 0.0}
	dummyVectorCol := entity.NewColumnFloatVector("dummy_vector", 4, [][]float32{dummyVector})

	// ID is AutoID, so we don't pass it.
	// However, Milvus Insert returns the generated IDs.
	cols := []entity.Column{titleCol, contentCol, sourceURLCol, docTypeCol, metadataCol, createdAtCol, dummyVectorCol}

	ids, err := m.client.Insert(ctx, DocumentsCollection, "", cols...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert document: %w", err)
	}

	if ids.Len() == 0 {
		return 0, fmt.Errorf("failed to insert document: no ID returned")
	}

	// Assuming int64 ID
	idCol, ok := ids.(*entity.ColumnInt64)
	if !ok {
		return 0, fmt.Errorf("unexpected ID type returned")
	}

	return idCol.Data()[0], nil
}

// GetDocument retrieves a document by ID
func (m *MilvusClient) GetDocument(id int64) (*Document, error) {
	ctx := context.Background()

	res, err := m.client.Query(ctx, DocumentsCollection, []string{}, fmt.Sprintf("id == %d", id), []string{"id", "title", "content", "source_url", "doc_type", "metadata", "created_at"})
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if res.GetColumn("id").Len() == 0 {
		return nil, fmt.Errorf("document not found")
	}

	doc := &Document{
		ID:        id,
		Title:     mustGetString(res, "title", 0),
		Content:   mustGetString(res, "content", 0),
		SourceURL: mustGetString(res, "source_url", 0),
		DocType:   mustGetString(res, "doc_type", 0),
		Metadata:  mustGetString(res, "metadata", 0),
		CreatedAt: mustGetInt64(res, "created_at", 0),
	}

	return doc, nil
}

// ListDocuments lists documents with pagination
func (m *MilvusClient) ListDocuments(limit, offset int) ([]Document, int64, error) {
	ctx := context.Background()

	res, err := m.client.Query(ctx, DocumentsCollection, []string{}, "id > 0", []string{"id", "title", "content", "source_url", "doc_type", "metadata", "created_at"}, client.WithLimit(int64(limit)), client.WithOffset(int64(offset)))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list documents: %w", err)
	}

	count := res.GetColumn("id").Len()
	docs := make([]Document, count)

	for i := 0; i < count; i++ {
		docs[i] = Document{
			ID:        mustGetInt64(res, "id", i),
			Title:     mustGetString(res, "title", i),
			Content:   mustGetString(res, "content", i),
			SourceURL: mustGetString(res, "source_url", i),
			DocType:   mustGetString(res, "doc_type", i),
			Metadata:  mustGetString(res, "metadata", i),
			CreatedAt: mustGetInt64(res, "created_at", i),
		}
	}

	// Total count is hard to get efficiently in Milvus without a separate counter or Count() query which might be slow.
	// For now returning count of current page or just -1 if unknown.
	// Let's try to get total count.
	countRes, err := m.client.Query(ctx, DocumentsCollection, []string{}, "id > 0", []string{"count(*)"})
	var total int64
	if err == nil && countRes.GetColumn("count(*)").Len() > 0 {
		total = countRes.GetColumn("count(*)").(*entity.ColumnInt64).Data()[0]
	}

	return docs, total, nil
}

// DeleteDocument deletes a document and its chunks
func (m *MilvusClient) DeleteDocument(id int64) error {
	ctx := context.Background()

	// Delete document
	if err := m.client.Delete(ctx, DocumentsCollection, "", fmt.Sprintf("id == %d", id)); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Delete chunks
	if err := m.client.Delete(ctx, CollectionName, "", fmt.Sprintf("document_id == %d", id)); err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	return nil
}

// Helper functions for extracting data from columns
func mustGetString(rs client.ResultSet, fieldName string, row int) string {
	col := rs.GetColumn(fieldName)
	if col == nil {
		return ""
	}
	val, err := col.Get(row)
	if err != nil {
		return ""
	}
	return val.(string)
}

func mustGetInt64(rs client.ResultSet, fieldName string, row int) int64 {
	col := rs.GetColumn(fieldName)
	if col == nil {
		return 0
	}
	val, err := col.Get(row)
	if err != nil {
		return 0
	}
	return val.(int64)
}
