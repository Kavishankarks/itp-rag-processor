package llm

import "context"

// LLMProvider defines the interface for LLM interactions
type LLMProvider interface {
	GenerateContent(ctx context.Context, prompt string) (string, error)
	Close() error
}
