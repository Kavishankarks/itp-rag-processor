package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/document-hub/go-api/internal/models"
	"github.com/pgvector/pgvector-go"
)

// Search godoc
// @Summary Search documents
// @Tags search
// @Param q query string true "Search query"
// @Param type query string false "Search type: fulltext, semantic, hybrid" default(hybrid)
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

	searchType := c.Query("type", "hybrid") // fulltext, semantic, hybrid
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
		results = h.fullTextSearch(query, limit, minScore)
	case "semantic":
		results = h.semanticSearch(query, limit, minScore)
	case "hybrid":
		results = h.hybridSearch(query, limit, minScore)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid search type. Use 'fulltext', 'semantic', or 'hybrid'",
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

// fullTextSearch performs PostgreSQL full-text search
func (h *Handler) fullTextSearch(query string, limit int, minScore float64) []models.SearchResult {
	var results []models.SearchResult

	rows, err := h.db.Raw(`
		SELECT
			d.id, d.title, d.content, d.source_url, d.doc_type, d.metadata, d.created_at, d.updated_at,
			ts_rank(to_tsvector('english', d.content), plainto_tsquery('english', ?)) as rank,
			ts_headline('english', d.content, plainto_tsquery('english', ?),
				'MaxWords=50, MinWords=20') as snippet
		FROM documents d
		WHERE to_tsvector('english', d.content) @@ plainto_tsquery('english', ?)
			AND ts_rank(to_tsvector('english', d.content), plainto_tsquery('english', ?)) >= ?
		ORDER BY rank DESC
		LIMIT ?
	`, query, query, query, query, minScore, limit).Rows()

	if err != nil {
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var doc models.Document
		var score float64
		var snippet string
		var metadata []byte

		err := rows.Scan(
			&doc.ID, &doc.Title, &doc.Content, &doc.SourceURL, &doc.DocType,
			&metadata, &doc.CreatedAt, &doc.UpdatedAt, &score, &snippet,
		)
		if err != nil {
			continue
		}
		doc.Metadata = metadata

		results = append(results, models.SearchResult{
			Document: doc,
			Score:    score,
			Snippet:  snippet,
		})
	}

	return results
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

	queryEmbedding := pgvector.NewVector(embeddings[0])

	var results []models.SearchResult

	rows, err := h.db.Raw(`
		SELECT DISTINCT ON (dc.document_id)
			d.id, d.title, d.content, d.source_url, d.doc_type, d.metadata, d.created_at, d.updated_at,
			1 - (dc.embedding <=> ?) as similarity,
			dc.chunk_text as snippet
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.document_id
		WHERE (1 - (dc.embedding <=> ?)) >= ?
		ORDER BY dc.document_id, (dc.embedding <=> ?) ASC
		LIMIT ?
	`, queryEmbedding, queryEmbedding, minScore, queryEmbedding, limit).Rows()

	if err != nil {
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var doc models.Document
		var score float64
		var snippet string
		var metadata []byte

		err := rows.Scan(
			&doc.ID, &doc.Title, &doc.Content, &doc.SourceURL, &doc.DocType,
			&metadata, &doc.CreatedAt, &doc.UpdatedAt, &score, &snippet,
		)
		if err != nil {
			continue
		}
		doc.Metadata = metadata

		results = append(results, models.SearchResult{
			Document: doc,
			Score:    score,
			Snippet:  snippet,
		})
	}

	return results
}

// hybridSearch combines full-text and semantic search with weighted scores
func (h *Handler) hybridSearch(query string, limit int, minScore float64) []models.SearchResult {
	// Get embedding for the query
	embeddings, err := h.embeddingClient.GetEmbeddings([]string{query})
	if err != nil {
		return []models.SearchResult{}
	}

	if len(embeddings) == 0 {
		return []models.SearchResult{}
	}

	queryEmbedding := pgvector.NewVector(embeddings[0])

	var results []models.SearchResult

	// Hybrid search: combine full-text (40%) and semantic (60%) scores
	rows, err := h.db.Raw(`
		WITH semantic_scores AS (
			SELECT DISTINCT ON (dc.document_id)
				dc.document_id,
				1 - (dc.embedding <=> ?) as semantic_score,
				dc.chunk_text as snippet
			FROM document_chunks dc
			ORDER BY dc.document_id, (dc.embedding <=> ?) ASC
		),
		fulltext_scores AS (
			SELECT
				d.id as document_id,
				ts_rank(to_tsvector('english', d.content), plainto_tsquery('english', ?)) as fulltext_score
			FROM documents d
			WHERE to_tsvector('english', d.content) @@ plainto_tsquery('english', ?)
		)
		SELECT
			d.id, d.title, d.content, d.source_url, d.doc_type, d.metadata, d.created_at, d.updated_at,
			(COALESCE(fs.fulltext_score, 0) * 0.4 + COALESCE(ss.semantic_score, 0) * 0.6) as hybrid_score,
			COALESCE(ss.snippet,
				ts_headline('english', d.content, plainto_tsquery('english', ?),
					'MaxWords=50, MinWords=20')) as snippet
		FROM documents d
		LEFT JOIN semantic_scores ss ON ss.document_id = d.id
		LEFT JOIN fulltext_scores fs ON fs.document_id = d.id
		WHERE (ss.semantic_score IS NOT NULL OR fs.fulltext_score IS NOT NULL)
			AND (COALESCE(fs.fulltext_score, 0) * 0.4 + COALESCE(ss.semantic_score, 0) * 0.6) >= ?
		ORDER BY hybrid_score DESC
		LIMIT ?
	`, queryEmbedding, queryEmbedding, query, query, query, minScore, limit).Rows()

	if err != nil {
		fmt.Println("Hybrid search error:", err)
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var doc models.Document
		var score float64
		var snippet string
		var metadata []byte

		err := rows.Scan(
			&doc.ID, &doc.Title, &doc.Content, &doc.SourceURL, &doc.DocType,
			&metadata, &doc.CreatedAt, &doc.UpdatedAt, &score, &snippet,
		)
		if err != nil {
			continue
		}
		doc.Metadata = metadata

		results = append(results, models.SearchResult{
			Document: doc,
			Score:    score,
			Snippet:  snippet,
		})
	}

	return results
}
