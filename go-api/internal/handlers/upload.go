package handlers

import (
	"fmt"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/document-hub/go-api/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
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

	// 3. Create document record
	doc := models.Document{
		Title:    file.Filename,
		Content:  normalized,
		DocType:  filepath.Ext(file.Filename),
		Metadata: datatypes.JSON([]byte(`{"source": "upload", "original_filename": "` + file.Filename + `"}`)),
	}

	// Start transaction
	tx := h.db.Begin()

	if err := tx.Create(&doc).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create document record",
		})
	}

	// 4. Chunk text
	chunks, err := h.embeddingClient.ChunkText(normalized, 500)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to chunk text: %v", err),
		})
	}

	// 5. Generate embeddings
	embeddings, err := h.embeddingClient.GetEmbeddings(chunks)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate embeddings: %v", err),
		})
	}

	// 6. Store chunks
	for i, chunk := range chunks {
		chunkRecord := models.DocumentChunk{
			DocumentID: doc.ID,
			ChunkText:  chunk,
			ChunkIndex: i,
			Embedding:  pgvector.NewVector(embeddings[i]),
		}
		if err := tx.Create(&chunkRecord).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create chunk",
			})
		}
	}

	tx.Commit()

	return c.Status(fiber.StatusCreated).JSON(doc)
}
