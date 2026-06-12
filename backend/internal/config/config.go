package config

import (
	"log"
	"os"
)

type Config struct {
	LLMMode string

	LLMAPIURL string
	LLMAPIKey string
	LLMModel  string

	ImageAPIURL string
	ImageAPIKey string
	ImageModel  string
}

func Load() *Config {
	cfg := &Config{
		LLMMode:     envOrDefault("LLM_MODE", "mock"),
		LLMAPIURL:   os.Getenv("LLM_API_URL"),
		LLMAPIKey:   os.Getenv("LLM_API_KEY"),
		LLMModel:    envOrDefault("LLM_MODEL", "deepseek-chat"),
		ImageAPIURL: os.Getenv("IMAGE_API_URL"),
		ImageAPIKey: os.Getenv("IMAGE_API_KEY"),
		ImageModel:  envOrDefault("IMAGE_MODEL", "dall-e-3"),
	}

	log.Printf("[CONFIG] LLM_MODE=%s", cfg.LLMMode)
	log.Printf("[CONFIG] LLM_API_URL=%s", cfg.LLMAPIURL)
	log.Printf("[CONFIG] LLM_API_KEY=%s", cfg.LLMAPIKey)
	log.Printf("[CONFIG] LLM_MODEL=%s", cfg.LLMModel)

	return cfg
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}



