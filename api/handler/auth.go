package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	authapi "dwizzyBRAIN/api/auth"
)

const (
	authStateCookieName    = "dwizzy_discord_oauth_state"
	authRedirectCookieName = "dwizzy_post_auth_redirect"
	authAccessCookieName   = "dwizzy_access_token"
	authRefreshCookieName  = "dwizzy_refresh_token"
)

type AuthHandler struct {
	service *authapi.Service
}

func NewAuthHandler(service *authapi.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/auth/discord/start", h.start)
	mux.HandleFunc("POST /v1/auth/discord/start", h.start)
	mux.HandleFunc("GET /v1/auth/discord/callback", h.callback)
	mux.HandleFunc("POST /v1/auth/web3/nonce", h.web3Nonce)
	mux.HandleFunc("POST /v1/auth/web3/verify", h.web3Verify)
	mux.HandleFunc("POST /v1/auth/refresh", h.refresh)
	mux.HandleFunc("POST /v1/auth/logout", h.logout)
	mux.HandleFunc("GET /v1/auth/me", h.me)
	mux.HandleFunc("GET /v1/auth/entitlement", h.entitlement)
}

func (h *AuthHandler) start(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.DiscordEnabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "discord oauth is unavailable")
		return
	}

	start, err := h.service.StartDiscordAuth(r.Context())
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authStateCookieName,
		Value:    start.State,
		Path:     "/v1/auth/discord",
		Domain:   h.service.CookieDomain(),
		HttpOnly: true,
		Secure:   h.service.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   max(1, int(time.Until(start.StateExpiresAt).Seconds())),
	})
	if nextPath := sanitizeNextPath(r.URL.Query().Get("next")); nextPath != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     authRedirectCookieName,
			Value:    nextPath,
			Path:     "/v1/auth/discord",
			Domain:   h.service.CookieDomain(),
			HttpOnly: true,
			Secure:   h.service.CookieSecure(),
			SameSite: http.SameSiteLaxMode,
			MaxAge:   max(1, int(time.Until(start.StateExpiresAt).Seconds())),
		})
	}

	if r.Method == http.MethodGet {
		http.Redirect(w, r, start.AuthorizationURL, http.StatusFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": start})
}

func (h *AuthHandler) callback(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.DiscordEnabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "discord oauth is unavailable")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	cookieState, _ := r.Cookie(authStateCookieName)
	redirectCookie, _ := r.Cookie(authRedirectCookieName)
	cookieValue := ""
	if cookieState != nil {
		cookieValue = cookieState.Value
	}
	redirectPath := ""
	if redirectCookie != nil {
		redirectPath = sanitizeNextPath(redirectCookie.Value)
	}

	result, err := h.service.CompleteDiscordAuth(r.Context(), code, state, cookieValue, authapi.RequestMeta{
		RemoteAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	h.setAuthCookies(w, result)
	clearCookie(w, authStateCookieName, "/v1/auth/discord", h.service.CookieDomain(), h.service.CookieSecure())
	clearCookie(w, authRedirectCookieName, "/v1/auth/discord", h.service.CookieDomain(), h.service.CookieSecure())

	if frontendOrigin := h.service.FrontendOrigin(); frontendOrigin != "" {
		targetPath := redirectPath
		if targetPath == "" {
			targetPath = "/dashboard"
		}
		target := frontendOrigin + targetPath
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *AuthHandler) web3Nonce(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "web3 auth is unavailable")
		return
	}

	var req authapi.Web3NonceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	result, err := h.service.RequestWeb3Nonce(r.Context(), req.WalletAddress, req.Purpose)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *AuthHandler) web3Verify(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "web3 auth is unavailable")
		return
	}

	var req authapi.Web3VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	result, err := h.service.VerifyWeb3Signature(r.Context(), req.WalletAddress, req.Purpose, req.Nonce, req.Signature, authapi.RequestMeta{
		RemoteAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	h.setAuthCookies(w, result)
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "discord oauth is unavailable")
		return
	}

	token := tokenFromRequest(r, authRefreshCookieName)
	if token == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		token = strings.TrimSpace(body.RefreshToken)
	}

	result, err := h.service.Refresh(r.Context(), token, authapi.RequestMeta{
		RemoteAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	h.setAuthCookies(w, result)
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "discord oauth is unavailable")
		return
	}

	token := tokenFromRequest(r, authRefreshCookieName)
	if token == "" {
		token = tokenFromRequest(r, authAccessCookieName)
	}
	if token == "" {
		var body struct {
			Token string `json:"token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		token = strings.TrimSpace(body.Token)
	}

	if err := h.service.Logout(r.Context(), token); err != nil {
		writeErrorFromErr(w, err)
		return
	}

	clearCookie(w, authAccessCookieName, "/", h.service.CookieDomain(), h.service.CookieSecure())
	clearCookie(w, authRefreshCookieName, "/", h.service.CookieDomain(), h.service.CookieSecure())
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true}})
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "discord oauth is unavailable")
		return
	}

	token := tokenFromRequest(r, authAccessCookieName)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "access token is required")
		return
	}

	result, err := h.service.Me(r.Context(), token)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *AuthHandler) entitlement(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "discord oauth is unavailable")
		return
	}

	token := tokenFromRequest(r, authAccessCookieName)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "access token is required")
		return
	}

	result, err := h.service.Entitlement(r.Context(), token)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, result authapi.AuthResponse) {
	http.SetCookie(w, &http.Cookie{
		Name:     authAccessCookieName,
		Value:    result.AccessToken,
		Path:     "/",
		Domain:   h.service.CookieDomain(),
		HttpOnly: true,
		Secure:   h.service.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   max(1, int(time.Until(result.AccessTokenExpiresAt).Seconds())),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     authRefreshCookieName,
		Value:    result.RefreshToken,
		Path:     "/",
		Domain:   h.service.CookieDomain(),
		HttpOnly: true,
		Secure:   h.service.CookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   max(1, int(time.Until(result.RefreshTokenExpiresAt).Seconds())),
	})
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

func clearCookie(w http.ResponseWriter, name, path, domain string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   strings.TrimSpace(domain),
		HttpOnly: true,
		Secure:   secure,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func sanitizeNextPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return ""
	}
	return value
}
