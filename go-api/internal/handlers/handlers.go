package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/document-hub/go-api/internal/embedding_client"
	"github.com/kavishankarks/document-hub/go-api/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type Handler struct {
	db              *gorm.DB
	embeddingClient *embedding_client.EmbeddingClient
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:              db,
		embeddingClient: embedding_client.NewClient(),
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

	// Check if document with same title already exists
	var existingDoc models.Document
	if err := h.db.Where("title = ?", req.Title).First(&existingDoc).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":       "Document with this title already exists",
			"existing_id": existingDoc.ID,
			"hint":        "Use PUT /api/v1/documents/:id to update the existing document",
		})
	}

	// Create document
	doc := models.Document{
		Title:     req.Title,
		Content:   req.Content,
		SourceURL: req.SourceURL,
		DocType:   req.DocType,
		Metadata:  req.Metadata,
	}

	// Start transaction
	tx := h.db.Begin()

	if err := tx.Create(&doc).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create document",
		})
	}

	// Chunk the content
	chunks, err := h.embeddingClient.ChunkText(req.Content, 500)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to chunk text: %v", err),
		})
	}

	// Get embeddings for all chunks
	embeddings, err := h.embeddingClient.GetEmbeddings(chunks)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate embeddings: %v", err),
		})
	}

	// Create chunk records
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

	var doc models.Document
	if err := h.db.Preload("Chunks").First(&doc, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve document",
		})
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

	var documents []models.Document
	var total int64

	h.db.Model(&models.Document{}).Count(&total)

	if err := h.db.Offset(skip).Limit(limit).Find(&documents).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve documents",
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
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	var req models.UpdateDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var doc models.Document
	if err := h.db.First(&doc, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve document",
		})
	}

	// Update fields
	if req.Title != nil {
		doc.Title = *req.Title
	}
	if req.Content != nil {
		doc.Content = *req.Content
		// If content changed, re-chunk and re-embed
		// This is a simplified version - in production you'd want more sophisticated logic
	}
	if req.SourceURL != nil {
		doc.SourceURL = *req.SourceURL
	}
	if req.DocType != nil {
		doc.DocType = *req.DocType
	}
	if len(req.Metadata) > 0 {
		doc.Metadata = req.Metadata
	}

	if err := h.db.Save(&doc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update document",
		})
	}

	return c.JSON(doc)
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

	result := h.db.Delete(&models.Document{}, id)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete document",
		})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
