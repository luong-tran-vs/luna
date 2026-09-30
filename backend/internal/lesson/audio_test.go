package lesson

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestAudioEndpoint(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	const id = "66f9a1b2c3d4e5f6a7b8c9d0"
	dir := filepath.Join(a.env.dir, id, "2")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0.mp3"), []byte("ID3data"), 0o600); err != nil {
		t.Fatal(err)
	}

	rec := a.do(t, http.MethodGet, "/api/audio/"+id+"/2/0", "learner", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "ID3data" {
		t.Fatalf("serve: %d %q", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "audio/mpeg" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "private, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", cc)
	}

	if rec := a.do(t, http.MethodGet, "/api/audio/"+id+"/2/0", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
	for _, path := range []string{
		"/api/audio/" + id + "/2/1",     // missing file
		"/api/audio/notanid/2/0",        // malformed id
		"/api/audio/" + id + "/-1/0",    // negative revision
		"/api/audio/" + id + "/2/x",     // non-numeric index
		"/api/audio/" + id + "/2/0.mp3", // extension not accepted
		"/api/audio/..%2F..%2Fetc/2/0",  // traversal attempt
	} {
		if rec := a.do(t, http.MethodGet, path, "admin", ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, rec.Code)
		}
	}
}
