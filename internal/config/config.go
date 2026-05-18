package config

import (
	"os"
	"strings"
)

type Config struct {
	Port       string
	SQLitePath string
	Debug      bool
}

func Load() Config {
	return Config{
		Port:       getEnv("PORT", "8080"),
		SQLitePath: getEnv("SQLITE_PATH", "gateway.db"),
		Debug:      parseBool(getEnv("DEBUG", "false")),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
