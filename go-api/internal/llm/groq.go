package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// GroqProvider implements LLMProvider for Groq API
type GroqProvider struct {
	apiKey string
	client *http.Client
	model  string
}

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewGroqProvider creates a new Groq provider
func NewGroqProvider() (*GroqProvider, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GROQ_API_KEY environment variable not set")
	}

	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = "openai/gpt-oss-20b" // Default model
	}

	return &GroqProvider{
		apiKey: apiKey,
		client: &http.Client{},
		model:  model,
	}, nil
}

// GenerateContent generates text content based on the prompt using Groq API
func (p *GroqProvider) GenerateContent(ctx context.Context, prompt string) (string, error) {
	reqBody := groqRequest{
		Model: p.model,
		Messages: []groqMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var groqResp groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("groq api error: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	return groqResp.Choices[0].Message.Content, nil
}

// Close is a no-op for GroqProvider as it uses http.Client
func (p *GroqProvider) Close() error {
	return nil
}
