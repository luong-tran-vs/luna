package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/config"
)

func envFrom(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoad(t *testing.T) {
	t.Parallel()

	const uriValue = "mongodb://leak-check.invalid:27017"

	tests := []struct {
		name      string
		env       map[string]string
		want      config.Config
		wantErrs  []string
		forbidden string
	}{
		{
			name: "defaults applied when only MONGO_URI is set",
			env:  map[string]string{"MONGO_URI": "mongodb://localhost:27017"},
			want: config.Config{
				DBDriver: "mongo", MongoURI: "mongodb://localhost:27017", MongoDatabase: "luna",
				HTTPAddr: ":8080", LogLevel: "info", CookieSecure: false,
				AIProvider: "gemini", GeminiModel: "gemini-3.5-flash-lite",
				DictionaryPath: "./data/dictionary/dictionary.db",
			},
		},
		{
			name: "explicit values override defaults",
			env: map[string]string{
				"MONGO_URI": "mongodb://x", "MONGO_DATABASE": "luna_test", "HTTP_ADDR": ":9000",
				"LOG_LEVEL": "debug", "COOKIE_SECURE": "TRUE",
				"AI_PROVIDER": "NONE", "GEMINI_API_KEY": "k", "GEMINI_MODEL": "gemini-x",
				"DICTIONARY_PATH": "/data/dictionary/dictionary.db",
			},
			want: config.Config{
				DBDriver: "mongo", MongoURI: "mongodb://x", MongoDatabase: "luna_test",
				HTTPAddr: ":9000", LogLevel: "debug", CookieSecure: true,
				AIProvider: "none", GeminiAPIKey: "k", GeminiModel: "gemini-x",
				DictionaryPath: "/data/dictionary/dictionary.db",
			},
		},
		{
			name:     "invalid COOKIE_SECURE",
			env:      map[string]string{"MONGO_URI": "mongodb://x", "COOKIE_SECURE": "yes"},
			wantErrs: []string{"COOKIE_SECURE must be true or false"},
		},
		{
			name:     "invalid AI_PROVIDER",
			env:      map[string]string{"MONGO_URI": "mongodb://x", "AI_PROVIDER": "openai"},
			wantErrs: []string{"AI_PROVIDER must be gemini or none"},
		},
		{
			name:      "API key never appears in errors",
			env:       map[string]string{"GEMINI_API_KEY": "secret-key-123", "AI_PROVIDER": "bad"},
			wantErrs:  []string{"MONGO_URI is required", "AI_PROVIDER"},
			forbidden: "secret-key-123",
		},
		{
			name:     "missing MONGO_URI",
			env:      map[string]string{},
			wantErrs: []string{"MONGO_URI is required"},
		},
		{
			name:     "blank MONGO_URI counts as missing",
			env:      map[string]string{"MONGO_URI": "   "},
			wantErrs: []string{"MONGO_URI is required"},
		},
		{
			name:      "invalid LOG_LEVEL",
			env:       map[string]string{"MONGO_URI": uriValue, "LOG_LEVEL": "verbose"},
			wantErrs:  []string{"LOG_LEVEL must be one of debug, info, warn, error"},
			forbidden: uriValue,
		},
		{
			name: "mysql with a DSN only: defaults for the rest, and the driver name is case-insensitive",
			env:  map[string]string{"DB_DRIVER": " MySQL ", "MYSQL_DSN": "luna:pw@tcp(db:3306)/luna"},
			want: config.Config{
				DBDriver: "mysql", MongoDatabase: "luna",
				MySQL: config.MySQL{
					DSN: "luna:pw@tcp(db:3306)/luna", Host: "localhost", Port: "3306", Database: "luna",
					MaxOpenConns: 20, MaxIdleConns: 5, ConnMaxLifetime: 5 * time.Minute, DialTimeout: 2 * time.Second,
				},
				HTTPAddr: ":8080", LogLevel: "info", AIProvider: "gemini", GeminiModel: "gemini-3.5-flash-lite",
				DictionaryPath: "./data/dictionary/dictionary.db",
			},
		},
		{
			name: "mysql from its parts and tuned pool",
			env: map[string]string{
				"DB_DRIVER": "mysql", "MYSQL_HOST": "db.internal", "MYSQL_PORT": "3307", "MYSQL_USER": "luna",
				"MYSQL_PASSWORD": " p@ss word ", "MYSQL_DATABASE": "luna_prod",
				"MYSQL_MAX_OPEN_CONNS": "50", "MYSQL_MAX_IDLE_CONNS": "10",
				"MYSQL_CONN_MAX_LIFETIME": "30m", "MYSQL_DIAL_TIMEOUT": "500ms",
			},
			want: config.Config{
				DBDriver: "mysql", MongoDatabase: "luna",
				MySQL: config.MySQL{
					Host: "db.internal", Port: "3307", User: "luna", Password: " p@ss word ", Database: "luna_prod",
					MaxOpenConns: 50, MaxIdleConns: 10, ConnMaxLifetime: 30 * time.Minute, DialTimeout: 500 * time.Millisecond,
				},
				HTTPAddr: ":8080", LogLevel: "info", AIProvider: "gemini", GeminiModel: "gemini-3.5-flash-lite",
				DictionaryPath: "./data/dictionary/dictionary.db",
			},
		},
		{
			name: "MYSQL_* are ignored, and not validated, when the database is mongo",
			env: map[string]string{
				"MONGO_URI": "mongodb://x", "MYSQL_PORT": "nope", "MYSQL_MAX_OPEN_CONNS": "-3",
			},
			want: config.Config{
				DBDriver: "mongo", MongoURI: "mongodb://x", MongoDatabase: "luna",
				HTTPAddr: ":8080", LogLevel: "info", AIProvider: "gemini", GeminiModel: "gemini-3.5-flash-lite",
				DictionaryPath: "./data/dictionary/dictionary.db",
			},
		},
		{
			name:     "mysql with neither a DSN nor a user, even with MONGO_URI set",
			env:      map[string]string{"DB_DRIVER": "mysql", "MONGO_URI": "mongodb://x"},
			wantErrs: []string{"MYSQL_USER (or MYSQL_DSN) is required when DB_DRIVER is mysql"},
		},
		{
			name: "invalid mysql numbers and durations are all reported",
			env: map[string]string{
				"DB_DRIVER": "mysql", "MYSQL_USER": "luna", "MYSQL_PORT": "70000",
				"MYSQL_MAX_OPEN_CONNS": "0", "MYSQL_MAX_IDLE_CONNS": "many",
				"MYSQL_CONN_MAX_LIFETIME": "5", "MYSQL_DIAL_TIMEOUT": "-1s",
			},
			wantErrs: []string{
				"MYSQL_PORT must be a number from 1 to 65535",
				"MYSQL_MAX_OPEN_CONNS must be a whole number of at least 1",
				"MYSQL_MAX_IDLE_CONNS must be a whole number of at least 1",
				"MYSQL_CONN_MAX_LIFETIME must be a duration",
				"MYSQL_DIAL_TIMEOUT must be a duration",
			},
		},
		{
			name:     "unknown DB_DRIVER",
			env:      map[string]string{"DB_DRIVER": "postgres", "MONGO_URI": "mongodb://x"},
			wantErrs: []string{"DB_DRIVER must be mongo or mysql"},
		},
		{
			name:      "MYSQL_DSN never appears in errors",
			env:       map[string]string{"DB_DRIVER": "mysql", "MYSQL_DSN": "luna:hunter2@tcp(db)/luna", "LOG_LEVEL": "loud"},
			wantErrs:  []string{"LOG_LEVEL"},
			forbidden: "hunter2",
		},
		{
			name: "MYSQL_PASSWORD never appears in errors",
			env: map[string]string{
				"DB_DRIVER": "mysql", "MYSQL_USER": "luna", "MYSQL_PASSWORD": "hunter2", "MYSQL_PORT": "bad",
			},
			wantErrs:  []string{"MYSQL_PORT"},
			forbidden: "hunter2",
		},
		{
			name:     "all errors reported together",
			env:      map[string]string{"LOG_LEVEL": "loud"},
			wantErrs: []string{"MONGO_URI is required", "LOG_LEVEL must be one of"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.Load(envFrom(tt.env))

			if len(tt.wantErrs) == 0 {
				if err != nil {
					t.Fatalf("Load() unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("Load() = %+v, want %+v", got, tt.want)
				}
				return
			}

			if err == nil {
				t.Fatalf("Load() error = nil, want errors %q", tt.wantErrs)
			}
			for _, want := range tt.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Load() error %q does not contain %q", err, want)
				}
			}
			if tt.forbidden != "" && strings.Contains(err.Error(), tt.forbidden) {
				t.Errorf("Load() error leaks value %q", tt.forbidden)
			}
		})
	}
}
