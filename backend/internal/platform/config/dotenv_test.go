package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/platform/config"
)

func TestParseDotEnv(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"# comment",
		"",
		"MONGO_URI=mongodb://localhost:27017",
		"  LOG_LEVEL = debug  ",
		`QUOTED="a b # not a comment"`,
		"SINGLE='x=y'",
		"export COOKIE_SECURE=false",
		"EMPTY=",
		"TRAILING=value # comment",
	}, "\n")

	got, err := config.ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDotEnv: %v", err)
	}
	want := map[string]string{
		"MONGO_URI":     "mongodb://localhost:27017",
		"LOG_LEVEL":     "debug",
		"QUOTED":        "a b # not a comment",
		"SINGLE":        "x=y",
		"COOKIE_SECURE": "false",
		"EMPTY":         "",
		"TRAILING":      "value",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseDotEnvRejectsBadLine(t *testing.T) {
	t.Parallel()

	_, err := config.ParseDotEnv(strings.NewReader("MONGO_URI=x\nkhông có dấu bằng\n"))
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("err = %v, want error mentioning line 2", err)
	}
}

// Not parallel: t.Setenv changes the process environment.
func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "LUNA_TEST_FROM_FILE=file\nLUNA_TEST_ALREADY_SET=file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUNA_TEST_ALREADY_SET", "real")
	t.Setenv("LUNA_TEST_FROM_FILE", "")
	if err := os.Unsetenv("LUNA_TEST_FROM_FILE"); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.LoadDotEnv(path)
	if err != nil || !loaded {
		t.Fatalf("LoadDotEnv = %v, %v; want true, nil", loaded, err)
	}
	if got := os.Getenv("LUNA_TEST_FROM_FILE"); got != "file" {
		t.Errorf("LUNA_TEST_FROM_FILE = %q, want value from file", got)
	}
	if got := os.Getenv("LUNA_TEST_ALREADY_SET"); got != "real" {
		t.Errorf("LUNA_TEST_ALREADY_SET = %q, real environment must win", got)
	}

	loaded, err = config.LoadDotEnv(filepath.Join(dir, "missing.env"))
	if err != nil || loaded {
		t.Fatalf("missing file: LoadDotEnv = %v, %v; want false, nil", loaded, err)
	}
}
