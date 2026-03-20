package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authapi "dwizzyBRAIN/api/auth"
)

type fakeAuthService struct {
	responses    map[string]authapi.MeResponse
	entitlements map[string]authapi.EntitlementResponse
	err          error
}

func (f fakeAuthService) Me(ctx context.Context, token string) (authapi.MeResponse, error) {
	if f.err != nil {
		return authapi.MeResponse{}, f.err
	}
	if resp, ok := f.responses[token]; ok {
		return resp, nil
	}
	return authapi.MeResponse{}, authapi.ErrUnauthorized
}

func (f fakeAuthService) Entitlement(ctx context.Context, token string) (authapi.EntitlementResponse, error) {
	if f.err != nil {
		return authapi.EntitlementResponse{}, f.err
	}
	if resp, ok := f.entitlements[token]; ok {
		return resp, nil
	}
	if resp, ok := f.responses[token]; ok {
		return authapi.EntitlementResponse{Plan: resp.User.Plan}, nil
	}
	return authapi.EntitlementResponse{}, authapi.ErrUnauthorized
}

func TestAuthMiddlewareRequirePlan(t *testing.T) {
	mw := NewAuthMiddleware(fakeAuthService{
		responses: map[string]authapi.MeResponse{
			"free-token": {
				User: authapi.UserProfile{
					Plan: "free",
				},
			},
			"premium-token": {
				User: authapi.UserProfile{
					Plan: "premium",
				},
			},
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := mw.RequirePlan("premium", next)

	t.Run("free blocked", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer free-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("premium allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer premium-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestAuthMiddlewareAuthenticateUnauthorized(t *testing.T) {
	mw := NewAuthMiddleware(fakeAuthService{err: errors.New("boom")})
	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected auth failure, got %d", rec.Code)
	}
}
