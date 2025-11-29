package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/document-hub/go-api/internal/models"
)

// Search godoc
// @Summary Search documents
// @Tags search
// @Param q query string true "Search query"
// @Param type query string false "Search type: semantic" default(semantic)
// @Param limit query int false "Result limit" default(10)
// @Param min_score query float64 false "Minimum score threshold (0.0-1.0)" default(0.3)
// @Success 200 {object} map[string]interface{}
// @Router /search [get]
func (h *Handler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	searchType := c.Query("type", "semantic") // fulltext, semantic, hybrid
	limit := c.QueryInt("limit", 10)
	minScore := c.QueryFloat("min_score", 0.3) // Default minimum score: 30%

	if limit > 100 {
		limit = 100
	}

	// Ensure min_score is between 0 and 1
	if minScore < 0 {
		minScore = 0
	} else if minScore > 1 {
		minScore = 1
	}

	var results []models.SearchResult

	switch searchType {
	case "fulltext":
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Full-text search is not supported with Milvus storage",
		})
	case "semantic", "hybrid":
		// Hybrid is currently same as semantic since we don't have full-text
		results = h.semanticSearch(query, limit, minScore)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid search type. Use 'semantic'",
		})
	}

	return c.JSON(fiber.Map{
		"query":       query,
		"search_type": searchType,
		"min_score":   minScore,
		"results":     results,
		"count":       len(results),
	})
}

// semanticSearch performs vector similarity search
func (h *Handler) semanticSearch(query string, limit int, minScore float64) []models.SearchResult {
	// Get embedding for the query
	embeddings, err := h.embeddingClient.GetEmbeddings([]string{query})
	if err != nil {
		return []models.SearchResult{}
	}

	if len(embeddings) == 0 {
		return []models.SearchResult{}
	}

	// Search in Milvus
	milvusResults, err := h.milvusClient.Search(embeddings[0], limit, minScore)
	if err != nil {
		fmt.Printf("Milvus search error: %v\n", err)
		return []models.SearchResult{}
	}

	if len(milvusResults) == 0 {
		return []models.SearchResult{}
	}

	var results []models.SearchResult

	// Fetch document details for each result
	// Note: This could be optimized with a batch GetDocument if available
	for _, res := range milvusResults {
		milvusDoc, err := h.milvusClient.GetDocument(res.DocumentID)
		if err != nil {
			fmt.Printf("Warning: Failed to get document %d: %v\n", res.DocumentID, err)
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

		results = append(results, models.SearchResult{
			Document: doc,
			Score:    float64(res.Score),
			Snippet:  res.ChunkText,
		})
	}

	return results
}

// hybridSearch combines full-text and semantic search with weighted scores
func (h *Handler) hybridSearch(query string, limit int, minScore float64) []models.SearchResult {
	// Deprecated: Just alias to semantic search for now
	return h.semanticSearch(query, limit, minScore)
}
