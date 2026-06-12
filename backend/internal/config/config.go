package config

import "os"

type Config struct {
	LLMMode string

	LLMAPIURL   string
	LLMAPIKey   string
	LLMModel    string

	ImageAPIURL string
	ImageAPIKey string
	ImageModel  string
}

func Load() *Config {
	return &Config{
		LLMMode:     envOrDefault("LLM_MODE", "mock"),
		LLMAPIURL:   os.Getenv("LLM_API_URL"),
		LLMAPIKey:   os.Getenv("LLM_API_KEY"),
		LLMModel:    envOrDefault("LLM_MODEL", "gpt-4o"),
		ImageAPIURL: os.Getenv("IMAGE_API_URL"),
		ImageAPIKey: os.Getenv("IMAGE_API_KEY"),
		ImageModel:  envOrDefault("IMAGE_MODEL", "dall-e-3"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
