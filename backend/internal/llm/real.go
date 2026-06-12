package llm

import "errors"

type OpenAIClassifier struct {
	Endpoint string
	APIKey   string
	Model    string
}

func (c *OpenAIClassifier) Classify(sentence string) (bool, string, error) {
	return false, "", errors.New("real LLM not implemented yet")
}

type OpenAIRefiner struct {
	Endpoint string
	APIKey   string
	Model    string
}

func (r *OpenAIRefiner) Refine(dev, newSentence string) (string, error) {
	return "", errors.New("real LLM not implemented yet")
}

type OpenAIGenerator struct {
	Endpoint string
	APIKey   string
	Model    string
}

func (g *OpenAIGenerator) Generate(prompt string) (string, error) {
	return "", errors.New("real image generation not implemented yet")
}
