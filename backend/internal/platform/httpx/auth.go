package httpx

import (
	"context"
	"net/http"
)

// SessionCookieName is the cookie that carries the login session token.
const SessionCookieName = "luna_session"

// RoleAdmin is the role allowed through RequireAdmin.
const RoleAdmin = "admin"

// Principal is the logged-in user attached to a request.
type Principal struct {
	UserID string
	Email  string
	Role   string
}

type principalKey struct{}

// PrincipalFrom returns the principal stored by RequireAuth.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// WithPrincipal returns a copy of ctx carrying p (used by RequireAuth and tests).
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// RequireAuth resolves the session cookie through resolve and rejects the request with
// 401 when the cookie is missing or the session is not valid.
func RequireAuth(resolve func(ctx context.Context, token string) (Principal, error)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				writeUnauthenticated(w)
				return
			}
			p, err := resolve(r.Context(), cookie.Value)
			if err != nil {
				writeUnauthenticated(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

// RequireAdmin rejects non-admin principals with 403. It must run after RequireAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		if p.Role != RoleAdmin {
			WriteError(w, http.StatusForbidden, "forbidden", "Bạn không có quyền truy cập")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeUnauthenticated(w http.ResponseWriter) {
	WriteError(w, http.StatusUnauthorized, "unauthenticated", "Bạn cần đăng nhập")
}
