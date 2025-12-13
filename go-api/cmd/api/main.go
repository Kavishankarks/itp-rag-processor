//go:generate /Users/kavishankarks/go/bin/swag init -g cmd/api/main.go --parseDependency --parseInternal
package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	_ "github.com/kavishankarks/itp-rag-processor/go-api/docs"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/embedding_client"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/handlers"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/llm"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/pipeline"
	"github.com/kavishankarks/itp-rag-processor/go-api/internal/vector"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title Document Hub API
// @version 1.0
// @description A high-performance documentation platform with intelligent search using full-text and semantic vector search.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/kavishankarks/itp-rag-processor
// @contact.email support@documenthub.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8000
// @BasePath /api/v1
// @schemes http https

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: os.Getenv("APP_NAME"),
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// Serve frontend
	app.Static("/", "../frontend")

	// Swagger documentation
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "itp-rag-processor-api",
		})
	})

	// Initialize Milvus client
	milvusURL := os.Getenv("MILVUS_URL")
	milvusToken := os.Getenv("MILVUS_TOKEN")
	if milvusURL == "" || milvusToken == "" {
		log.Fatal("MILVUS_URL and MILVUS_TOKEN environment variables are required")
	}

	milvusClient, err := vector.Initialize(milvusURL, milvusToken)
	if err != nil {
		log.Fatal("Failed to initialize Milvus client:", err)
	}
	defer milvusClient.Close()

	if err := milvusClient.EnsureCollections(); err != nil {
		log.Fatal("Failed to ensure Milvus collection:", err)
	}

	// Initialize handlers
	h := handlers.NewHandler(milvusClient)

	// Initialize embedding client and pipeline orchestrator
	embeddingClient := embedding_client.NewClient()
	orchestrator := pipeline.NewOrchestrator(embeddingClient, milvusClient)
	pipelineHandler := handlers.NewPipelineHandler(orchestrator)

	// Initialize LLM provider
	var llmProvider llm.LLMProvider
	llmProviderType := os.Getenv("LLM_PROVIDER")

	if llmProviderType == "groq" {
		llmProvider, err = llm.NewGroqProvider()
		if err != nil {
			log.Printf("Warning: Failed to initialize Groq provider: %v", err)
		}
	} else {
		// Default to Gemini
		llmProvider, err = llm.NewGeminiProvider(context.Background())
		if err != nil {
			log.Printf("Warning: Failed to initialize Gemini provider: %v", err)
		}
	}

	generateHandler := handlers.NewGenerateHandler(llmProvider, h)

	// API routes
	api := app.Group("/api/v1")

	// Document routes (existing functionality)
	api.Post("/documents", h.CreateDocument)
	api.Post("/documents/upload", h.UploadDocument)
	api.Get("/documents/:id", h.GetDocument)
	api.Get("/documents", h.ListDocuments)
	api.Put("/documents/:id", h.UpdateDocument)
	api.Delete("/documents/:id", h.DeleteDocument)

	// Search routes (existing functionality)
	api.Get("/search", h.Search)

	// Generation routes (new)
	api.Post("/generate", generateHandler.Generate)

	// Pipeline routes (new RAG processing pipeline)
	api.Post("/pipeline/start", pipelineHandler.StartPipeline)
	api.Get("/pipeline/:id/status", pipelineHandler.GetPipelineStatus)
	api.Get("/pipeline/:id/results", pipelineHandler.GetPipelineResults)
	api.Post("/pipeline/:id/cancel", pipelineHandler.CancelPipeline)
	api.Get("/pipelines", pipelineHandler.ListPipelines)

	// Start server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("swagger docs at http://localhost:{port}/swagger/index.html")
	log.Printf("swagger redoc at http://localhost:{port}/swagger/redoc")
	if err := app.Listen(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
