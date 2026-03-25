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

	"dwizzyBRAIN/api"
	authapi "dwizzyBRAIN/api/auth"
	contentapi "dwizzyBRAIN/api/content"
	defiapi "dwizzyBRAIN/api/defi"
	downloadapi "dwizzyBRAIN/api/download"
	"dwizzyBRAIN/api/handler"
	newsapi "dwizzyBRAIN/api/news"
	samehadakuapi "dwizzyBRAIN/api/samehadaku"
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

	postgresPool, err := storage.NewPostgresPoolFromEnv(ctx)
	if err != nil {
		return err
	}
	defer postgresPool.Close()

	var authService *authapi.Service
	if cfg, err := authapi.ConfigFromEnv(); err != nil {
		log.Printf("discord oauth unavailable: %v", err)
	} else {
		authService = authapi.NewService(postgresPool, cfg)
		if resolver, err := authapi.NewSubscriptionResolverFromEnv(ctx, postgresPool); err != nil {
			log.Printf("subscription resolver unavailable: %v", err)
		} else if resolver != nil {
			authService.SetPlanResolver(resolver)
		}
	}
	authHandler := handler.NewAuthHandler(authService)
	defiService := defiapi.NewService(postgresPool)
	defiHandler := handler.NewDefiHandler(defiService)
	newsService := newsapi.NewService(postgresPool)
	newsHandler := handler.NewNewsHandler(newsService)
	contentBaseURL, contentUserAgent := contentapi.ConfigFromEnv()
	contentService := contentapi.NewHybridService(postgresPool, contentapi.NewHTTPClient(contentBaseURL, contentUserAgent, 15*time.Second), contentBaseURL)
	contentHandler := handler.NewContentHandler(contentService)
	downloadCfg, err := irag.ConfigFromEnv()
	if err != nil {
		log.Printf("download irag unavailable: %v", err)
	}
	downloadService := downloadapi.NewService(downloadCfg, nil, nil)
	downloadHandler := handler.NewDownloadHandler(downloadService)
	if downloadService.Enabled() {
		log.Printf("download irag service initialized")
	} else {
		log.Printf("download irag unavailable")
	}
	samehadakuService := samehadakuapi.NewService(postgresPool)
	if samehadakuCfg, err := samehadakuapi.ConfigFromEnv(); err != nil {
		log.Printf("samehadaku supabase unavailable: %v", err)
	} else if svc, err := samehadakuapi.NewSupabaseServiceFromConfig(ctx, samehadakuCfg, &http.Client{Timeout: 15 * time.Second}); err != nil {
		log.Printf("samehadaku supabase init failed: %v", err)
	} else {
		samehadakuService = svc
		log.Printf("samehadaku catalog using supabase backend")
	}
	samehadakuHandler := handler.NewSamehadakuHandler(samehadakuService)
	router := api.NewRouter(defiHandler, newsHandler, authHandler, contentHandler, downloadHandler, samehadakuHandler)

	port := resolvePort(os.Getenv("API_PORT"))
	server := newHTTPServer(port, router)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	return waitForServer(ctx, server.Shutdown, errCh)
}

func resolvePort(value string) string {
	if strings.TrimSpace(value) == "" {
		return "8080"
	}
	return strings.TrimSpace(value)
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func isExpectedServeError(err error) bool {
	return err == nil || errors.Is(err, http.ErrServerClosed)
}

func waitForServer(ctx context.Context, shutdown func(context.Context) error, errCh <-chan error) error {
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return shutdown(shutdownCtx)
	case err := <-errCh:
		if !isExpectedServeError(err) {
			return err
		}
		return nil
	}
}
