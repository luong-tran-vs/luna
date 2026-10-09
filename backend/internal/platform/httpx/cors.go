package httpx

import (
	"net/http"
	"slices"
)

// CORS lets a frontend served from one of origins call the API with the session cookie.
// "*" allows every origin: the caller's origin is echoed back, because browsers refuse a
// literal "*" on requests that carry cookies.
// Requests from other origins pass through without CORS headers, so the browser blocks them.
// With no origins it is a no-op (frontend and API share one origin, e.g. behind a reverse proxy).
func CORS(origins []string) Middleware {
	return func(next http.Handler) http.Handler {
		if len(origins) == 0 {
			return next
		}
		anyOrigin := slices.Contains(origins, "*")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			h := w.Header()
			h.Add("Vary", "Origin")
			if origin == "" || (!anyOrigin && !slices.Contains(origins, origin)) {
				next.ServeHTTP(w, r)
				return
			}

			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			// The export file name and the request id are read by the frontend.
			h.Set("Access-Control-Expose-Headers", "Content-Disposition, "+RequestIDHeader)

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
				h.Set("Access-Control-Allow-Headers", "Content-Type, "+RequestIDHeader)
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
