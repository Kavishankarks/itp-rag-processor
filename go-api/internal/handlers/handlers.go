package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/embedding_client"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/models"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/vector"
)

type Handler struct {
	embeddingClient *embedding_client.EmbeddingClient
	milvusClient    *vector.MilvusClient
}

func NewHandler(milvusClient *vector.MilvusClient) *Handler {
	return &Handler{
		embeddingClient: embedding_client.NewClient(),
		milvusClient:    milvusClient,
	}
}

// CreateDocument godoc
// @Summary Create document
// @Tags documents
// @Accept json
// @Produce json
// @Param body body models.CreateDocumentRequest true "Document"
// @Success 201 {object} models.Document
// @Failure 400,409,500 {object} map[string]interface{}
// @Router /documents [post]
func (h *Handler) CreateDocument(c *fiber.Ctx) error {
	var req models.CreateDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Create document in Milvus
	// Note: MilvusClient.CreateDocument checks for duplicates by title (if implemented)
	// or we rely on unique constraint if any.
	// Our CreateDocument implementation does check for existing title.

	metadataBytes, _ := json.Marshal(req.Metadata)

	milvusDoc := &vector.Document{
		Title:     req.Title,
		Content:   req.Content,
		SourceURL: req.SourceURL,
		DocType:   req.DocType,
		Metadata:  string(metadataBytes),
	}

	docID, err := h.milvusClient.CreateDocument(milvusDoc)
	if err != nil {
		// Check if error is duplicate
		// This depends on how CreateDocument returns error.
		// Assuming generic error for now, but we could improve this.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create document: %v", err),
		})
	}

	// Chunk the content
	chunks, err := h.embeddingClient.ChunkText(req.Content, 500)
	if err != nil {
		// Try to cleanup
		h.milvusClient.DeleteDocument(docID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to chunk text: %v", err),
		})
	}

	// Get embeddings for all chunks
	embeddings, err := h.embeddingClient.GetEmbeddings(chunks)
	if err != nil {
		h.milvusClient.DeleteDocument(docID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate embeddings: %v", err),
		})
	}

	// Create chunk records for Milvus
	var milvusChunks []vector.Chunk
	for i, chunk := range chunks {
		milvusChunks = append(milvusChunks, vector.Chunk{
			DocumentID: docID,
			ChunkIndex: int64(i),
			ChunkText:  chunk,
			Embedding:  embeddings[i],
		})
	}

	// Store chunks in Milvus
	if err := h.milvusClient.AddChunks(milvusChunks); err != nil {
		h.milvusClient.DeleteDocument(docID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to store chunks in vector DB: %v", err),
		})
	}

	// Construct response
	respDoc := models.Document{
		ID:        uint(docID),
		Title:     req.Title,
		Content:   req.Content,
		SourceURL: req.SourceURL,
		DocType:   req.DocType,
		Metadata:  req.Metadata,
	}

	return c.Status(fiber.StatusCreated).JSON(respDoc)
}

// GetDocument godoc
// @Summary Get document by ID
// @Tags documents
// @Param id path int true "Document ID"
// @Success 200 {object} models.Document
// @Failure 400,404,500 {object} map[string]string
// @Router /documents/{id} [get]
func (h *Handler) GetDocument(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	milvusDoc, err := h.milvusClient.GetDocument(int64(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
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

	return c.JSON(doc)
}

// ListDocuments godoc
// @Summary List documents
// @Tags documents
// @Param skip query int false "Skip"
// @Param limit query int false "Limit"
// @Success 200 {object} map[string]interface{}
// @Router /documents [get]
func (h *Handler) ListDocuments(c *fiber.Ctx) error {
	skip, _ := strconv.Atoi(c.Query("skip", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if limit > 100 {
		limit = 100
	}

	milvusDocs, total, err := h.milvusClient.ListDocuments(limit, skip)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve documents: %v", err),
		})
	}

	var documents []models.Document
	for _, md := range milvusDocs {
		var metadata map[string]interface{}
		json.Unmarshal([]byte(md.Metadata), &metadata)

		documents = append(documents, models.Document{
			ID:        uint(md.ID),
			Title:     md.Title,
			Content:   md.Content,
			SourceURL: md.SourceURL,
			DocType:   md.DocType,
			Metadata:  metadata,
		})
	}

	return c.JSON(fiber.Map{
		"documents": documents,
		"total":     total,
		"skip":      skip,
		"limit":     limit,
	})
}

// UpdateDocument godoc
// @Summary Update document
// @Tags documents
// @Param id path int true "Document ID"
// @Param body body models.UpdateDocumentRequest true "Updates"
// @Success 200 {object} models.Document
// @Failure 400,404,500 {object} map[string]string
// @Router /documents/{id} [put]
func (h *Handler) UpdateDocument(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "Update document is not supported with Milvus storage yet",
	})
}

// DeleteDocument godoc
// @Summary Delete document
// @Tags documents
// @Param id path int true "Document ID"
// @Success 204
// @Failure 400,404,500 {object} map[string]string
// @Router /documents/{id} [delete]
func (h *Handler) DeleteDocument(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	if err := h.milvusClient.DeleteDocument(int64(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete document: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
