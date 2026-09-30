package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// Handler serves /api/auth/*.
type Handler struct {
	svc          *Service
	cookieSecure bool
	log          *slog.Logger
}

// NewHandler returns a Handler. cookieSecure adds the Secure flag to the session cookie.
func NewHandler(svc *Service, cookieSecure bool, log *slog.Logger) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure, log: log}
}

type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Timezone string `json:"timezone"`
}

type userEnvelope struct {
	User userResponse `json:"user"`
}

func toResponse(u User) userEnvelope {
	return userEnvelope{User: userResponse{ID: u.ID, Email: u.Email, Role: u.Role, Timezone: u.Timezone}}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Timezone string `json:"timezone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles POST /api/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	u, token, err := h.svc.Register(r.Context(), req.Email, req.Password, req.Timezone)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.setSession(w, token)
	httpx.WriteJSON(w, http.StatusCreated, toResponse(u))
}

// Login handles POST /api/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if httpx.DecodeJSON(w, r, &req) != nil {
		return
	}
	u, token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.setSession(w, token)
	httpx.WriteJSON(w, http.StatusOK, toResponse(u))
}

// Logout handles POST /api/auth/logout. It always succeeds and clears the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(httpx.SessionCookieName); err == nil {
		if err := h.svc.Logout(r.Context(), c.Value); err != nil {
			h.log.ErrorContext(r.Context(), "logout failed", slog.Any("error", err))
		}
	}
	h.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me handles GET /api/auth/me. It must run behind httpx.RequireAuth.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p, ok := httpx.PrincipalFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Bạn cần đăng nhập")
		return
	}
	u, err := h.svc.UserByID(r.Context(), p.UserID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(u))
}

// ResolvePrincipal adapts Service.Authenticate for httpx.RequireAuth.
func (h *Handler) ResolvePrincipal(ctx context.Context, token string) (httpx.Principal, error) {
	u, err := h.svc.Authenticate(ctx, token)
	if err != nil {
		return httpx.Principal{}, err
	}
	return httpx.Principal{UserID: u.ID, Email: u.Email, Role: string(u.Role)}, nil
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	var locked *LockedError
	switch {
	case errors.As(err, &verr):
		httpx.WriteFieldErrors(w, verr.Fields)
	case errors.Is(err, ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", "Email này đã được dùng")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Email hoặc mật khẩu không đúng")
	case errors.Is(err, ErrUnauthenticated):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Bạn cần đăng nhập")
	case errors.As(err, &locked):
		seconds := int(math.Ceil(locked.RetryAfter.Seconds()))
		minutes := int(math.Ceil(locked.RetryAfter.Minutes()))
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		httpx.WriteJSON(w, http.StatusTooManyRequests, lockedBody{
			Error:             "account_locked",
			Message:           fmt.Sprintf("Đăng nhập tạm khoá. Thử lại sau %d phút", minutes),
			RetryAfterSeconds: seconds,
		})
	default:
		h.log.ErrorContext(r.Context(), "auth request failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Có lỗi xảy ra, vui lòng thử lại")
	}
}

type lockedBody struct {
	Error             string `json:"error"`
	Message           string `json:"message"`
	RetryAfterSeconds int    `json:"retryAfterSeconds"`
}

func (h *Handler) setSession(w http.ResponseWriter, token string) {
	h.writeCookie(w, token, int(SessionTTL.Seconds()))
}

func (h *Handler) clearSession(w http.ResponseWriter) {
	h.writeCookie(w, "", -1)
}

// writeCookie is the only place the session cookie is written. Secure comes from
// COOKIE_SECURE because the app is served over plain HTTP on localhost.
func (h *Handler) writeCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure is set from config (COOKIE_SECURE); HttpOnly and SameSite=Strict are always on
		Name:     httpx.SessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}
