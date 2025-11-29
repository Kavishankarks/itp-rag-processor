package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/llm"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/models"
)

// GenerateHandler handles LLM generation requests
type GenerateHandler struct {
	llmProvider   llm.LLMProvider
	searchHandler *Handler
}

// NewGenerateHandler creates a new generation handler
func NewGenerateHandler(llmProvider llm.LLMProvider, searchHandler *Handler) *GenerateHandler {
	return &GenerateHandler{
		llmProvider:   llmProvider,
		searchHandler: searchHandler,
	}
}

// GenerateRequest represents the request body for generation
type GenerateRequest struct {
	Prompt           string  `json:"prompt"`
	IncludeCitations bool    `json:"include_citations"`
	MinScore         float64 `json:"min_score"`
	Limit            int     `json:"limit"`
}

// GenerateResponse represents the generation response
type GenerateResponse struct {
	GeneratedText string                `json:"generated_text"`
	Sources       []models.SearchResult `json:"sources"`
}

// Generate godoc
// @Summary Generate content using LLM
// @Description Generates content based on the provided prompt and retrieved context from documents
// @Tags generation
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "Generation request"
// @Success 200 {object} GenerateResponse
// @Failure 400,500 {object} map[string]string
// @Router /generate [post]
func (h *GenerateHandler) Generate(c *fiber.Ctx) error {
	if h.llmProvider == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "LLM provider not initialized. Please check your GEMINI_API_KEY configuration.",
		})
	}

	var req GenerateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Prompt is required",
		})
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 5
	}
	if req.MinScore == 0 {
		req.MinScore = 0.3
	}

	// 1. Retrieve relevant context using Hybrid Search
	// We access the search logic directly from the existing handler
	results := h.searchHandler.hybridSearch(req.Prompt, req.Limit, req.MinScore)

	// 2. Construct the prompt
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Context information is below.\n---------------------\n")

	for i, result := range results {
		contextBuilder.WriteString(fmt.Sprintf("Source %d (Title: %s):\n%s\n\n", i+1, result.Document.Title, result.Snippet))
	}

	contextBuilder.WriteString("---------------------\n")
	contextBuilder.WriteString("Given the context information and not prior knowledge, answer the query.\n")
	if req.IncludeCitations {
		contextBuilder.WriteString("Please include citations to the sources used (e.g. [Source 1]).\n")
	}
	contextBuilder.WriteString(fmt.Sprintf("Query: %s\n", req.Prompt))
	contextBuilder.WriteString("Answer: ")

	// 3. Generate content
	generatedText, err := h.llmProvider.GenerateContent(context.Background(), contextBuilder.String())
	if err != nil {
		fmt.Printf("Error generating content: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate content: %v", err),
		})
	}

	// 4. Return response
	return c.JSON(GenerateResponse{
		GeneratedText: generatedText,
		Sources:       results,
	})
}
