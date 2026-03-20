package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dwizzyBRAIN/engine/storage"
	"dwizzyBRAIN/irag"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := irag.ConfigFromEnv()
	if err != nil {
		return err
	}

	var cache = irag.Cache(nil)
	if client, err := storage.NewValkeyClientFromEnv(ctx); err != nil {
		log.Printf("irag cache unavailable: %v", err)
	} else {
		defer client.Close()
		cache = irag.NewRedisCache(client)
	}

	var logs *irag.LogStore
	if pool, err := storage.NewPostgresPoolFromEnv(ctx); err != nil {
		log.Printf("irag logs unavailable: %v", err)
	} else {
		defer pool.Close()
		logs = irag.NewLogStore(pool)
	}

	service := irag.NewService(cfg, cache, logs)
	server := &http.Server{
		Addr:              ":" + resolvePort(os.Getenv("IRAG_PORT")),
		Handler:           irag.NewRouter(service),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("irag listening on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	return waitForServer(ctx, server.Shutdown, errCh)
}

func resolvePort(value string) string {
	if strings.TrimSpace(value) == "" {
		return "8081"
	}
	return strings.TrimSpace(value)
}

func waitForServer(ctx context.Context, shutdown func(context.Context) error, errCh <-chan error) error {
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return shutdown(shutdownCtx)
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
