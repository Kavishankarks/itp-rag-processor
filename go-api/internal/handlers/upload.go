package handlers

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/models"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/vector"
)

// UploadDocument godoc
// @Summary Upload and process a document
// @Description Uploads a file (PDF, Doc, Word, PPT, HTML), converts it to markdown, normalizes, chunks, embeds, and stores it.
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file"
// @Success 201 {object} models.Document
// @Failure 400,500 {object} map[string]string
// @Router /documents/upload [post]
func (h *Handler) UploadDocument(c *fiber.Ctx) error {
	// Get file from request
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file provided",
		})
	}

	// Open file
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open file",
		})
	}
	defer f.Close()

	// 1. Convert file to markdown
	markdown, err := h.embeddingClient.ConvertDocument(file.Filename, f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to convert document: %v", err),
		})
	}

	// 2. Normalize text
	normalized, err := h.embeddingClient.NormalizeText(markdown, true)
	if err != nil {
		// Log warning but continue with original markdown if normalization fails
		fmt.Printf("Warning: Normalization failed: %v\n", err)
		normalized = markdown
	}

	// 3. Create document record in Milvus
	metadata := map[string]interface{}{
		"source":            "upload",
		"original_filename": file.Filename,
	}
	metadataBytes, _ := json.Marshal(metadata)

	milvusDoc := &vector.Document{
		Title:     file.Filename,
		Content:   "", // Store empty content to avoid size limits. Chunks contain the actual content.
		SourceURL: "", // No source URL for uploaded files
		DocType:   filepath.Ext(file.Filename),
		Metadata:  string(metadataBytes),
	}

	docID, err := h.milvusClient.CreateDocument(milvusDoc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create document record: %v", err),
		})
	}

	// 4. Chunk text
	chunks, err := h.embeddingClient.ChunkText(normalized, 500)
	if err != nil {
		h.milvusClient.DeleteDocument(docID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to chunk text: %v", err),
		})
	}

	// 5. Generate embeddings
	embeddings, err := h.embeddingClient.GetEmbeddings(chunks)
	if err != nil {
		h.milvusClient.DeleteDocument(docID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate embeddings: %v", err),
		})
	}

	// 6. Store chunks
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
		ID:       uint(docID),
		Title:    file.Filename,
		Content:  normalized,
		DocType:  filepath.Ext(file.Filename),
		Metadata: metadata,
	}

	return c.Status(fiber.StatusCreated).JSON(respDoc)
}
