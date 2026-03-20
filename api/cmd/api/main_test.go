package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestResolvePort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "8080"},
		{name: "spaces", input: "   ", want: "8080"},
		{name: "trimmed", input: " 9090 ", want: "9090"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolvePort(tc.input); got != tc.want {
				t.Fatalf("resolvePort(%q)=%q want=%q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	srv := newHTTPServer("9090", handler)

	if srv.Addr != ":9090" {
		t.Fatalf("Addr=%q want=:9090", srv.Addr)
	}
	if srv.Handler != handler {
		t.Fatalf("Handler mismatch")
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout=%v want=5s", srv.ReadHeaderTimeout)
	}
}

func TestIsExpectedServeError(t *testing.T) {
	t.Parallel()

	if !isExpectedServeError(nil) {
		t.Fatal("nil should be expected")
	}
	if !isExpectedServeError(http.ErrServerClosed) {
		t.Fatal("http.ErrServerClosed should be expected")
	}
	if isExpectedServeError(errors.New("boom")) {
		t.Fatal("arbitrary error should not be expected")
	}
}

func TestWaitForServerErrorPath(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)
	errCh <- errors.New("boom")

	called := false
	err := waitForServer(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	}, errCh)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
	if called {
		t.Fatal("shutdown should not be called on immediate server error")
	}
}

func TestWaitForServerClosedPath(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)
	errCh <- http.ErrServerClosed

	err := waitForServer(context.Background(), func(ctx context.Context) error {
		return errors.New("should not be called")
	}, errCh)
	if err != nil {
		t.Fatalf("expected nil for ErrServerClosed, got %v", err)
	}
}

func TestWaitForServerShutdownPath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	errCh := make(chan error)
	expected := errors.New("shutdown-failed")
	called := false
	err := waitForServer(ctx, func(shutdownCtx context.Context) error {
		called = true
		return expected
	}, errCh)
	if !called {
		t.Fatal("expected shutdown to be called on context cancel")
	}
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
