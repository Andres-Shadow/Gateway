package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port        string
	SQLitePath  string
	Debug       bool
	AdminAuth   AdminAuthConfig
	CORS        CORSConfig
	RateLimit   RateLimitConfig
	Proxy       ProxyConfig
	HealthCheck HealthCheckConfig
}

type AdminAuthConfig struct {
	Username  string
	Password  string
	JWTSecret string
	TokenTTL  time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type RateLimitConfig struct {
	Enabled           bool
	RequestsPerMinute int
}

type ProxyConfig struct {
	MaxRetries            int
	RetryBackoff          time.Duration
	CircuitFailureLimit   int
	CircuitCooldownPeriod time.Duration
}

type HealthCheckConfig struct {
	Enabled  bool
	Interval time.Duration
	Timeout  time.Duration
}

func Load() Config {
	return Config{
		Port:       getEnv("PORT", "8080"),
		SQLitePath: getEnv("SQLITE_PATH", "gateway.db"),
		Debug:      parseBool(getEnv("DEBUG", "false")),
		AdminAuth: AdminAuthConfig{
			Username:  getEnv("ADMIN_USERNAME", "admin"),
			Password:  getEnv("ADMIN_PASSWORD", "admin"),
			JWTSecret: getEnv("ADMIN_JWT_SECRET", "change-me-in-production"),
			TokenTTL:  getDuration("ADMIN_TOKEN_TTL", 12*time.Hour),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
			AllowedMethods: splitCSV(getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")),
			AllowedHeaders: splitCSV(getEnv("CORS_ALLOWED_HEADERS", "Authorization,Content-Type")),
		},
		RateLimit: RateLimitConfig{
			Enabled:           parseBool(getEnv("RATE_LIMIT_ENABLED", "true")),
			RequestsPerMinute: getInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 120),
		},
		Proxy: ProxyConfig{
			MaxRetries:            getInt("PROXY_MAX_RETRIES", 2),
			RetryBackoff:          getDuration("PROXY_RETRY_BACKOFF", 100*time.Millisecond),
			CircuitFailureLimit:   getInt("CIRCUIT_FAILURE_LIMIT", 3),
			CircuitCooldownPeriod: getDuration("CIRCUIT_COOLDOWN", 30*time.Second),
		},
		HealthCheck: HealthCheckConfig{
			Enabled:  parseBool(getEnv("HEALTH_CHECK_ENABLED", "true")),
			Interval: getDuration("HEALTH_CHECK_INTERVAL", 30*time.Second),
			Timeout:  getDuration("HEALTH_CHECK_TIMEOUT", 2*time.Second),
		},
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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func getInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err == nil {
		return parsed
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
