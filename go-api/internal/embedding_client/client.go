package embedding_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type EmbeddingClient struct {
	baseURL    string
	httpClient *http.Client
}

type EmbeddingRequest struct {
	Texts []string `json:"texts"`
}

type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimension  int         `json:"dimension"`
}

type ChunkRequest struct {
	Text      string `json:"text"`
	ChunkSize int    `json:"chunk_size,omitempty"`
}

type ChunkResponse struct {
	Chunks []string `json:"chunks"`
}

type EnrichTopicRequest struct {
	TopicName  string `json:"topic_name"`
	MaxResults int    `json:"max_results,omitempty"`
}

type EnrichTopicResponse struct {
	TopicName       string                   `json:"topic_name"`
	SearchQuery     string                   `json:"search_query"`
	Results         []map[string]interface{} `json:"results"`
	CombinedContent string                   `json:"combined_content"`
	ResultCount     int                      `json:"result_count"`
}

type NormalizeRequest struct {
	Text          string `json:"text"`
	CleanHTMLTags bool   `json:"clean_html_tags"`
}

type NormalizeResponse struct {
	NormalizedText string                 `json:"normalized_text"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewClient creates a new embedding service client
func NewClient() *EmbeddingClient {
	baseURL := os.Getenv("EMBEDDING_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8002"
	}

	return &EmbeddingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetEmbeddings retrieves embeddings for the given texts
func (c *EmbeddingClient) GetEmbeddings(texts []string) ([][]float32, error) {
	reqBody := EmbeddingRequest{Texts: texts}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/v1/embeddings", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned %d: %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embResp.Embeddings, nil
}

// ChunkText splits text into chunks
func (c *EmbeddingClient) ChunkText(text string, chunkSize int) ([]string, error) {
	reqBody := ChunkRequest{
		Text:      text,
		ChunkSize: chunkSize,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/v1/chunk", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned %d: %s", resp.StatusCode, string(body))
	}

	var chunkResp ChunkResponse
	if err := json.NewDecoder(resp.Body).Decode(&chunkResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return chunkResp.Chunks, nil
}

// HealthCheck checks if the embedding service is healthy
func (c *EmbeddingClient) HealthCheck() error {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/health", c.baseURL))
	if err != nil {
		return fmt.Errorf("failed to reach embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("embedding service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// EnrichTopic enriches a curriculum topic with web search results
func (c *EmbeddingClient) EnrichTopic(topicName string, maxResults int) (map[string]interface{}, error) {
	reqBody := EnrichTopicRequest{
		TopicName:  topicName,
		MaxResults: maxResults,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/v1/enrich-topic", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned %d: %s", resp.StatusCode, string(body))
	}

	var enrichResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&enrichResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return enrichResp, nil
}

// NormalizeText normalizes text content
func (c *EmbeddingClient) NormalizeText(text string, cleanHTML bool) (string, error) {
	reqBody := NormalizeRequest{
		Text:          text,
		CleanHTMLTags: cleanHTML,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/v1/normalize", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("embedding service returned %d: %s", resp.StatusCode, string(body))
	}

	var normalizeResp NormalizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&normalizeResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return normalizeResp.NormalizedText, nil
}

type ConvertResponse struct {
	Markdown string `json:"markdown"`
	Filename string `json:"filename"`
}

// ConvertDocument converts a document to markdown
func (c *EmbeddingClient) ConvertDocument(filename string, content io.Reader) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, content); err != nil {
		return "", fmt.Errorf("failed to copy content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/v1/convert", c.baseURL),
		writer.FormDataContentType(),
		body,
	)
	if err != nil {
		return "", fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("embedding service returned %d: %s", resp.StatusCode, string(body))
	}

	var convertResp ConvertResponse
	if err := json.NewDecoder(resp.Body).Decode(&convertResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return convertResp.Markdown, nil
}
