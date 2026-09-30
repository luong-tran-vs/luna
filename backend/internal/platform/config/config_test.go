package config_test

import (
	"strings"
	"testing"

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
				MongoURI: "mongodb://localhost:27017", MongoDatabase: "luna",
				HTTPAddr: ":8080", LogLevel: "info", CookieSecure: false,
				TTSURL: "http://localhost:8880", TTSVoice: "af_heart", AudioDir: "./data/audio",
				AIProvider: "gemini", GeminiModel: "gemini-3.5-flash-lite",
				DictionaryPath: "./data/dictionary/dictionary.db",
			},
		},
		{
			name: "explicit values override defaults",
			env: map[string]string{
				"MONGO_URI": "mongodb://x", "MONGO_DATABASE": "luna_test", "HTTP_ADDR": ":9000",
				"LOG_LEVEL": "debug", "COOKIE_SECURE": "TRUE",
				"TTS_URL": "http://kokoro:8880", "TTS_VOICE": "bf_emma", "AUDIO_DIR": "/data/audio",
				"AI_PROVIDER": "NONE", "GEMINI_API_KEY": "k", "GEMINI_MODEL": "gemini-x",
				"DICTIONARY_PATH": "/data/dictionary/dictionary.db",
			},
			want: config.Config{
				MongoURI: "mongodb://x", MongoDatabase: "luna_test",
				HTTPAddr: ":9000", LogLevel: "debug", CookieSecure: true,
				TTSURL: "http://kokoro:8880", TTSVoice: "bf_emma", AudioDir: "/data/audio",
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
			name:     "invalid TTS_URL",
			env:      map[string]string{"MONGO_URI": "mongodb://x", "TTS_URL": "kokoro:8880"},
			wantErrs: []string{"TTS_URL must be an http or https URL"},
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
