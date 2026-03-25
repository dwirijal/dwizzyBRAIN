package handler

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type fakeDownloadService struct {
	enabled   bool
	status    int
	body      string
	path      string
	rawQuery  string
	callCount int32
}

func (f *fakeDownloadService) Enabled() bool {
	return f.enabled
}

func (f *fakeDownloadService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&f.callCount, 1)
	f.path = r.URL.Path
	f.rawQuery = r.URL.RawQuery
	if f.status == 0 {
		f.status = http.StatusOK
	}
	w.WriteHeader(f.status)
	_, _ = w.Write([]byte(f.body))
}

func TestDownloadHandlerDelegatesToService(t *testing.T) {
	t.Parallel()

	svc := &fakeDownloadService{
		enabled: true,
		status:  http.StatusCreated,
		body:    "downloaded",
	}
	mux := http.NewServeMux()
	NewDownloadHandler(svc).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/download/youtube/audio?url=https://example.com/watch?v=1&quality=high", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected delegated status %d, got %d", http.StatusCreated, rec.Code)
	}
	if got := rec.Body.String(); got != "downloaded" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := atomic.LoadInt32(&svc.callCount); got != 1 {
		t.Fatalf("expected one call, got %d", got)
	}
	if svc.path != "/v1/download/youtube/audio" {
		t.Fatalf("expected path to be preserved, got %q", svc.path)
	}
	if svc.rawQuery != "url=https://example.com/watch?v=1&quality=high" {
		t.Fatalf("expected query to be preserved, got %q", svc.rawQuery)
	}
}

func TestDownloadHandlerReturnsServiceUnavailableWhenDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		service downloadReader
	}{
		{name: "nil service", service: nil},
		{name: "disabled service", service: &fakeDownloadService{enabled: false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewDownloadHandler(tt.service).Register(mux)

			req := httptest.NewRequest(http.MethodGet, "/v1/download/spotify", nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected 503, got %d", rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected json content-type, got %q", got)
			}
		})
	}
}
