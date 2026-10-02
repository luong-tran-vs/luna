// Package config loads the backend configuration from environment variables.
package config

import (
	"errors"
	"slices"
	"strings"
)

// Config holds every setting the backend reads at startup.
type Config struct {
	MongoURI      string
	MongoDatabase string
	HTTPAddr      string
	LogLevel      string
	CookieSecure  bool

	AIProvider   string
	GeminiAPIKey string
	GeminiModel  string

	DictionaryPath string
}

var logLevels = []string{"debug", "info", "warn", "error"}

// Load reads the configuration through getenv (usually os.Getenv).
// It reports every problem at once and never includes variable values in errors.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		MongoURI:       strings.TrimSpace(getenv("MONGO_URI")),
		MongoDatabase:  valueOr(getenv("MONGO_DATABASE"), "luna"),
		HTTPAddr:       valueOr(getenv("HTTP_ADDR"), ":8080"),
		LogLevel:       strings.ToLower(valueOr(getenv("LOG_LEVEL"), "info")),
		AIProvider:     strings.ToLower(valueOr(getenv("AI_PROVIDER"), "gemini")),
		GeminiAPIKey:   strings.TrimSpace(getenv("GEMINI_API_KEY")),
		GeminiModel:    valueOr(getenv("GEMINI_MODEL"), "gemini-3.5-flash-lite"),
		DictionaryPath: valueOr(getenv("DICTIONARY_PATH"), "./data/dictionary/dictionary.db"),
	}

	var errs []error
	if cfg.MongoURI == "" {
		errs = append(errs, errors.New("config: MONGO_URI is required"))
	}
	if !slices.Contains(logLevels, cfg.LogLevel) {
		errs = append(errs, errors.New("config: LOG_LEVEL must be one of debug, info, warn, error"))
	}
	switch strings.ToLower(valueOr(getenv("COOKIE_SECURE"), "false")) {
	case "true":
		cfg.CookieSecure = true
	case "false":
	default:
		errs = append(errs, errors.New("config: COOKIE_SECURE must be true or false"))
	}
	if cfg.AIProvider != "gemini" && cfg.AIProvider != "none" {
		errs = append(errs, errors.New("config: AI_PROVIDER must be gemini or none"))
	}
	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func valueOr(v, fallback string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return fallback
}
