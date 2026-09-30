package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// LoadDotEnv reads KEY=VALUE lines from path into the process environment, for local
// development only. Variables that are already set win over the file. A missing file is
// not an error; the returned bool reports whether the file was read.
func LoadDotEnv(path string) (bool, error) {
	f, err := os.Open(path) //nolint:gosec // path is a fixed developer file, not user input
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("config: open %s: %w", path, err)
	}
	defer f.Close() //nolint:errcheck // read-only file

	vars, err := ParseDotEnv(f)
	if err != nil {
		return false, fmt.Errorf("config: %s: %w", path, err)
	}
	for k, v := range vars {
		if _, set := os.LookupEnv(k); set {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return false, fmt.Errorf("config: set %s: %w", k, err)
		}
	}
	return true, nil
}

// ParseDotEnv parses a .env file: blank lines and # comments are skipped, an optional
// "export " prefix is allowed, values may be wrapped in single or double quotes, and an
// unquoted value ends at " #".
func ParseDotEnv(r io.Reader) (map[string]string, error) {
	vars := map[string]string{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", n)
		}
		vars[key] = parseValue(strings.TrimSpace(value))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return vars, nil
}

func parseValue(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
