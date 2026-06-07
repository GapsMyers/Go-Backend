package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config stores all environment-driven application settings.
type Config struct {
	AppPort          string
	AppEnv           string
	DatabaseURL      string
	JWTSecret        string
	JWTExpireMinutes int
	AllowedOrigins   []string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	ShutdownTimeout  time.Duration
}

// Load reads configuration from environment variables (and optional .env file).
func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		AppPort:          getEnv("APP_PORT", "8080"),
		AppEnv:           getEnv("APP_ENV", "development"),
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:        strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTExpireMinutes: getEnvInt("JWT_EXPIRE_MINUTES", 60),
		AllowedOrigins:   splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:*,http://127.0.0.1:*")),
		ReadTimeout:      time.Duration(getEnvInt("READ_TIMEOUT_SECONDS", 10)) * time.Second,
		WriteTimeout:     time.Duration(getEnvInt("WRITE_TIMEOUT_SECONDS", 15)) * time.Second,
		ShutdownTimeout:  time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", 10)) * time.Second,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	if cfg.JWTExpireMinutes <= 0 {
		return Config{}, errors.New("JWT_EXPIRE_MINUTES must be greater than 0")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	if len(origins) == 0 {
		origins = append(origins, "*")
	}

	return origins
}
