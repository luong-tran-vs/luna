package lesson

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
)

var objectIDPattern = regexp.MustCompile(`^[0-9a-f]{24}$`)

// AudioHandler serves GET /api/audio/{lessonId}/{revision}/{index} from dir. Path values are
// parsed strictly (hex id, non-negative integers), so no user text reaches the file path.
func AudioHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("lessonId")
		revision, revErr := strconv.Atoi(r.PathValue("revision"))
		index, idxErr := strconv.Atoi(r.PathValue("index"))
		if !objectIDPattern.MatchString(id) || revErr != nil || idxErr != nil || revision < 0 || index < 0 {
			http.NotFound(w, r)
			return
		}

		file := filepath.Join(dir, id, strconv.Itoa(revision), strconv.Itoa(index)+".mp3")
		w.Header().Set("Content-Type", "audio/mpeg")
		// The URL contains the content revision, so a given URL never changes.
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		http.ServeFile(w, r, file) //nolint:gosec // file is built only from a validated hex id and two integers
	})
}
