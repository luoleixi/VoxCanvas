package config

import (
	"log"
	"os"

	"voxcanvas/backend/internal/logger"
)

type Config struct {
	LLMMode string

	ChatAPIURL string
	ChatModel  string

	ImageAPIURL string
	ImageModel  string

	APIKey string // single key for all services
}

func Load() *Config {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}

	cfg := &Config{
		LLMMode:     envOrDefault("LLM_MODE", "mock"),
		ChatAPIURL:  envOrDefault("CHAT_API_URL", "https://dashscope.aliyuncs.com/compatible-mode"),
		ChatModel:   envOrDefault("CHAT_MODEL", "qwen-plus"),
		ImageAPIURL: envOrDefault("IMAGE_API_URL", "https://dashscope.aliyuncs.com"),
		ImageModel:  envOrDefault("IMAGE_MODEL", "qwen-image-2.0-pro"),
		APIKey:      apiKey,
	}

	log.Printf("[CONFIG] LLM_MODE=%s", cfg.LLMMode)
	log.Printf("[CONFIG] CHAT_API_URL=%s", cfg.ChatAPIURL)
	log.Printf("[CONFIG] CHAT_MODEL=%s", cfg.ChatModel)
	log.Printf("[CONFIG] IMAGE_API_URL=%s", cfg.ImageAPIURL)
	log.Printf("[CONFIG] IMAGE_MODEL=%s", cfg.ImageModel)
	log.Printf("[CONFIG] API_KEY=%s", logger.MaskSecret(cfg.APIKey))

	return cfg
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
