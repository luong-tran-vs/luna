// Package config loads the backend configuration from environment variables.
package config

import (
	"errors"
	"slices"
	"strings"
)

// Config holds every setting the backend reads at startup.
type Config struct {
	// DBDriver picks the database: "mongo" (default) or "mysql".
	DBDriver      string
	MongoURI      string
	MongoDatabase string
	// MySQL is read only when DBDriver is "mysql"; it is the zero value otherwise.
	MySQL        MySQL
	HTTPAddr     string
	LogLevel     string
	CookieSecure bool
	// CORSOrigins are the web origins (scheme://host[:port]) allowed to call the API from a
	// browser, for a frontend served from another origin. Empty means same-origin only.
	CORSOrigins []string

	AIProvider   string
	GeminiAPIKey string
	GeminiModel  string
	// GeminiImageModel draws the vocabulary pictures (F23).
	GeminiImageModel string

	DictionaryPath string
}

var logLevels = []string{"debug", "info", "warn", "error"}

// Load reads the configuration through getenv (usually os.Getenv).
// It reports every problem at once and never includes variable values in errors.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DBDriver:         strings.ToLower(valueOr(getenv("DB_DRIVER"), "mongo")),
		MongoURI:         strings.TrimSpace(getenv("MONGO_URI")),
		MongoDatabase:    valueOr(getenv("MONGO_DATABASE"), "luna"),
		HTTPAddr:         valueOr(getenv("HTTP_ADDR"), ":8080"),
		LogLevel:         strings.ToLower(valueOr(getenv("LOG_LEVEL"), "info")),
		AIProvider:       strings.ToLower(valueOr(getenv("AI_PROVIDER"), "gemini")),
		GeminiAPIKey:     strings.TrimSpace(getenv("GEMINI_API_KEY")),
		GeminiModel:      valueOr(getenv("GEMINI_MODEL"), "gemini-3.5-flash-lite"),
		GeminiImageModel: valueOr(getenv("GEMINI_IMAGE_MODEL"), "gemini-2.5-flash-image"),
		DictionaryPath:   valueOr(getenv("DICTIONARY_PATH"), "./data/dictionary/dictionary.db"),
		CORSOrigins:      splitList(getenv("CORS_ORIGINS")),
	}

	var errs []error
	// Only the setting of the chosen database is required.
	switch cfg.DBDriver {
	case "mongo":
		if cfg.MongoURI == "" {
			errs = append(errs, errors.New("config: MONGO_URI is required"))
		}
	case "mysql":
		var mysqlErrs []error
		cfg.MySQL, mysqlErrs = loadMySQL(getenv)
		errs = append(errs, mysqlErrs...)
	default:
		errs = append(errs, errors.New("config: DB_DRIVER must be mongo or mysql"))
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

// splitList reads a comma-separated list, dropping blanks and trailing slashes; nil when empty.
func splitList(v string) []string {
	var out []string
	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimRight(strings.TrimSpace(item), "/"); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func valueOr(v, fallback string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return fallback
}
