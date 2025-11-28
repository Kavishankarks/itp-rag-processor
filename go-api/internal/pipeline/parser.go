package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kavishankarks/document-hub/go-api/internal/models"
	"gopkg.in/yaml.v3"
)

// CurriculumParser handles parsing of curriculum data from various formats
type CurriculumParser struct{}

// NewCurriculumParser creates a new curriculum parser
func NewCurriculumParser() *CurriculumParser {
	return &CurriculumParser{}
}

// ParseJSON parses curriculum from JSON string
func (p *CurriculumParser) ParseJSON(data string) (*models.Curriculum, error) {
	var curriculum models.Curriculum
	if err := json.Unmarshal([]byte(data), &curriculum); err != nil {
		return nil, fmt.Errorf("failed to parse JSON curriculum: %w", err)
	}

	if err := p.validate(&curriculum); err != nil {
		return nil, err
	}

	return &curriculum, nil
}

// ParseYAML parses curriculum from YAML string
func (p *CurriculumParser) ParseYAML(data string) (*models.Curriculum, error) {
	var curriculum models.Curriculum
	if err := yaml.Unmarshal([]byte(data), &curriculum); err != nil {
		return nil, fmt.Errorf("failed to parse YAML curriculum: %w", err)
	}

	if err := p.validate(&curriculum); err != nil {
		return nil, err
	}

	return &curriculum, nil
}

// ParseMarkdown parses curriculum from markdown-style text
// Expected format:
// # Course Title
// ## Module: Module Name
// - Topic 1
// - Topic 2
func (p *CurriculumParser) ParseMarkdown(data string) (*models.Curriculum, error) {
	lines := strings.Split(data, "\n")

	var curriculum models.Curriculum
	var currentModule *models.CurriculumModule

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Course title (# Title)
		if strings.HasPrefix(line, "# ") {
			curriculum.Title = strings.TrimPrefix(line, "# ")
			continue
		}

		// Module name (## Module: Name or ## Name)
		if strings.HasPrefix(line, "## ") {
			moduleName := strings.TrimPrefix(line, "## ")
			moduleName = strings.TrimPrefix(moduleName, "Module: ")

			if currentModule != nil {
				curriculum.Modules = append(curriculum.Modules, *currentModule)
			}

			currentModule = &models.CurriculumModule{
				Name:   moduleName,
				Topics: []string{},
			}
			continue
		}

		// Topics (- Topic or * Topic)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if currentModule != nil {
				topic := strings.TrimPrefix(line, "- ")
				topic = strings.TrimPrefix(topic, "* ")
				currentModule.Topics = append(currentModule.Topics, topic)
			}
			continue
		}
	}

	// Add the last module
	if currentModule != nil {
		curriculum.Modules = append(curriculum.Modules, *currentModule)
	}

	if err := p.validate(&curriculum); err != nil {
		return nil, err
	}

	return &curriculum, nil
}

// ParseAuto automatically detects format and parses
func (p *CurriculumParser) ParseAuto(data string) (*models.Curriculum, error) {
	data = strings.TrimSpace(data)

	// Try JSON first
	if strings.HasPrefix(data, "{") {
		return p.ParseJSON(data)
	}

	// Try YAML
	if strings.Contains(data, "title:") || strings.Contains(data, "modules:") {
		return p.ParseYAML(data)
	}

	// Default to markdown
	return p.ParseMarkdown(data)
}

// validate checks if curriculum is valid
func (p *CurriculumParser) validate(curriculum *models.Curriculum) error {
	if curriculum.Title == "" {
		return fmt.Errorf("curriculum title is required")
	}

	if len(curriculum.Modules) == 0 {
		return fmt.Errorf("curriculum must have at least one module")
	}

	for i, module := range curriculum.Modules {
		if module.Name == "" {
			return fmt.Errorf("module %d: name is required", i)
		}
		if len(module.Topics) == 0 {
			return fmt.Errorf("module %s: must have at least one topic", module.Name)
		}
	}

	return nil
}

// ExtractAllTopics extracts all topics from curriculum
func (p *CurriculumParser) ExtractAllTopics(curriculum *models.Curriculum) []string {
	var topics []string
	for _, module := range curriculum.Modules {
		topics = append(topics, module.Topics...)
	}
	return topics
}

// GenerateTopicContext generates context string for a topic
func (p *CurriculumParser) GenerateTopicContext(
	curriculum *models.Curriculum,
	topic string,
) string {
	// Find which module this topic belongs to
	var moduleName string
	var moduleDescription string

	for _, module := range curriculum.Modules {
		for _, t := range module.Topics {
			if t == topic {
				moduleName = module.Name
				moduleDescription = module.Description
				break
			}
		}
		if moduleName != "" {
			break
		}
	}

	context := fmt.Sprintf("Course: %s\n", curriculum.Title)
	if moduleName != "" {
		context += fmt.Sprintf("Module: %s\n", moduleName)
		if moduleDescription != "" {
			context += fmt.Sprintf("Description: %s\n", moduleDescription)
		}
	}
	context += fmt.Sprintf("Topic: %s\n\n", topic)

	return context
}
