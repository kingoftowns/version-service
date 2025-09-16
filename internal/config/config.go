package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	RedisURL    string
	GitRepoURL  string
	GitUsername string
	GitToken    string
	GitBranch   string
	LogLevel    string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		GitRepoURL:  getEnv("GIT_REPO_URL", ""),
		GitUsername: getEnv("GIT_USERNAME", "version-service"),
		GitToken:    getEnv("GIT_TOKEN", ""),
		GitBranch:   getEnv("GIT_BRANCH", "main"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	if cfg.GitRepoURL == "" {
		return nil, fmt.Errorf("GIT_REPO_URL is required")
	}

	if cfg.GitToken == "" {
		return nil, fmt.Errorf("GIT_TOKEN is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}