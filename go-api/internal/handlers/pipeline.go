package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kavishankarks/document-hub/go-api/internal/models"
	"github.com/kavishankarks/document-hub/go-api/internal/pipeline"
)

// PipelineHandler handles pipeline-related requests
type PipelineHandler struct {
	orchestrator *pipeline.Orchestrator
}

// NewPipelineHandler creates a new pipeline handler
func NewPipelineHandler(orchestrator *pipeline.Orchestrator) *PipelineHandler {
	return &PipelineHandler{
		orchestrator: orchestrator,
	}
}

// StartPipeline starts a new pipeline run
// @Summary Start a new RAG pipeline
// @Description Starts a new pipeline to process course curriculum with web search, normalization, chunking, and embedding
// @Tags pipeline
// @Accept json
// @Produce json
// @Param request body models.StartPipelineRequest true "Pipeline configuration"
// @Success 201 {object} models.PipelineRun
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/pipeline/start [post]
func (h *PipelineHandler) StartPipeline(c *fiber.Ctx) error {
	var req models.StartPipelineRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Start the pipeline
	pipelineRun, err := h.orchestrator.StartPipeline(&req.Curriculum, req.Config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(pipelineRun)
}

// GetPipelineStatus gets the status of a pipeline run
// @Summary Get pipeline status
// @Description Retrieves the current status and progress of a pipeline run
// @Tags pipeline
// @Produce json
// @Param id path int true "Pipeline Run ID"
// @Success 200 {object} models.PipelineStatusResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/pipeline/{id}/status [get]
func (h *PipelineHandler) GetPipelineStatus(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pipeline ID",
		})
	}

	status, err := h.orchestrator.GetPipelineStatus(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(status)
}

// GetPipelineResults gets the results of a completed pipeline run
// @Summary Get pipeline results
// @Description Retrieves the documents and chunks created by a pipeline run
// @Tags pipeline
// @Produce json
// @Param id path int true "Pipeline Run ID"
// @Success 200 {object} models.PipelineResultsResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/pipeline/{id}/results [get]
func (h *PipelineHandler) GetPipelineResults(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pipeline ID",
		})
	}

	results, err := h.orchestrator.GetPipelineResults(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(results)
}

// CancelPipeline cancels a running pipeline
// @Summary Cancel pipeline
// @Description Cancels a running pipeline execution
// @Tags pipeline
// @Produce json
// @Param id path int true "Pipeline Run ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/pipeline/{id}/cancel [post]
func (h *PipelineHandler) CancelPipeline(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pipeline ID",
		})
	}

	if err := h.orchestrator.CancelPipeline(uint(id)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Pipeline cancelled successfully",
	})
}

// ListPipelines lists all pipeline runs with pagination
// @Summary List pipeline runs
// @Description Retrieves a paginated list of all pipeline runs
// @Tags pipeline
// @Produce json
// @Param skip query int false "Number of records to skip" default(0)
// @Param limit query int false "Maximum number of records to return" default(20)
// @Param status query string false "Filter by status (pending, processing, completed, failed)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pipelines [get]
func (h *PipelineHandler) ListPipelines(c *fiber.Ctx) error {
	skip := c.QueryInt("skip", 0)
	limit := c.QueryInt("limit", 20)
	// status := c.Query("status", "") // Status filtering not implemented in in-memory store yet

	// Limit maximum
	if limit > 100 {
		limit = 100
	}

	pipelineRuns, total, err := h.orchestrator.ListPipelines(limit, skip)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch pipeline runs",
		})
	}

	return c.JSON(fiber.Map{
		"total":   total,
		"skip":    skip,
		"limit":   limit,
		"results": pipelineRuns,
	})
}
