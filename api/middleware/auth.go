package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	authapi "dwizzyBRAIN/api/auth"
)

const (
	accessCookieName = "dwizzy_access_token"
)

type AuthService interface {
	Me(context.Context, string) (authapi.MeResponse, error)
	Entitlement(context.Context, string) (authapi.EntitlementResponse, error)
}

type Principal struct {
	Me authapi.MeResponse
}

type contextKey struct{}

type AuthMiddleware struct {
	auth AuthService
}

func NewAuthMiddleware(auth AuthService) *AuthMiddleware {
	if auth == nil {
		return nil
	}
	return &AuthMiddleware{auth: auth}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if m == nil || m.auth == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusServiceUnavailable, "service_unavailable", "auth middleware is unavailable")
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := tokenFromRequest(r, accessCookieName)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "access token is required")
			return
		}

		result, err := m.auth.Me(r.Context(), token)
		if err != nil {
			writeErrorFromErr(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, Principal{Me: result})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequirePlan(required string, next http.Handler) http.Handler {
	if m == nil || m.auth == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusServiceUnavailable, "service_unavailable", "auth middleware is unavailable")
		})
	}
	required = normalizePlan(required)
	if required == "" || required == "free" {
		return m.Authenticate(next)
	}
	if next == nil {
		next = http.NotFoundHandler()
	}
	return m.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "access token is required")
			return
		}
		if plan, err := m.livePlan(r); err == nil {
			if normalizePlan(plan) != required {
				writeError(w, http.StatusForbidden, "forbidden", "premium access is required")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if normalizePlan(principal.Me.User.Plan) != required {
			writeError(w, http.StatusForbidden, "forbidden", "premium access is required")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func normalizePlan(plan string) string {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case "premium":
		return "premium"
	default:
		return "free"
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	value := ctx.Value(contextKey{})
	principal, ok := value.(Principal)
	return principal, ok
}

func tokenFromRequest(r *http.Request, cookieName string) string {
	if cookie, err := r.Cookie(cookieName); err == nil && cookie != nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return ""
}

func (m *AuthMiddleware) livePlan(r *http.Request) (string, error) {
	if m == nil || m.auth == nil {
		return "", nil
	}
	token := tokenFromRequest(r, accessCookieName)
	if token == "" {
		return "", nil
	}
	result, err := m.auth.Entitlement(r.Context(), token)
	if err != nil {
		return "", err
	}
	return result.Plan, nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeErrorFromErr(w http.ResponseWriter, err error) {
	if err == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "unexpected error")
		return
	}

	if errors.Is(err, authapi.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	if errors.Is(err, authapi.ErrInvalidState) || errors.Is(err, authapi.ErrInvalidToken) || errors.Is(err, authapi.ErrInvalidNonce) || errors.Is(err, authapi.ErrInvalidWalletAddress) || errors.Is(err, authapi.ErrUnsupportedPurpose) {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if errors.Is(err, authapi.ErrTokenExpired) || errors.Is(err, authapi.ErrTokenReuse) || errors.Is(err, authapi.ErrSessionRevoked) || errors.Is(err, authapi.ErrNonceExpired) || errors.Is(err, authapi.ErrNonceConsumed) || errors.Is(err, authapi.ErrInvalidSignature) {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
}
