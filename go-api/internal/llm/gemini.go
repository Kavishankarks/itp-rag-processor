package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiProvider implements LLMProvider for Google Gemini
type GeminiProvider struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(ctx context.Context) (*GeminiProvider, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-3-pro-preview")

	// Set default safety settings to be less restrictive for educational content
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
	}

	return &GeminiProvider{
		client: client,
		model:  model,
	}, nil
}

// GenerateContent generates text content based on the prompt
func (p *GeminiProvider) GenerateContent(ctx context.Context, prompt string) (string, error) {
	resp, err := p.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			result += string(txt)
		}
	}

	return result, nil
}

// Close closes the Gemini client
func (p *GeminiProvider) Close() error {
	return p.client.Close()
}
