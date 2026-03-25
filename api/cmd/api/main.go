package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dwizzyBRAIN/api"
	authapi "dwizzyBRAIN/api/auth"
	defiapi "dwizzyBRAIN/api/defi"
	downloadapi "dwizzyBRAIN/api/download"
	"dwizzyBRAIN/api/handler"
	marketapi "dwizzyBRAIN/api/market"
	"dwizzyBRAIN/api/middleware"
	newsapi "dwizzyBRAIN/api/news"
	quantapi "dwizzyBRAIN/api/quant"
	"dwizzyBRAIN/engine/market/coingecko"
	"dwizzyBRAIN/engine/market/ohlcv"
	engticker "dwizzyBRAIN/engine/market/ticker"
	"dwizzyBRAIN/engine/storage"
	"dwizzyBRAIN/irag"

	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
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

	var cache redis.Cmdable
	if client, err := storage.NewValkeyClientFromEnv(ctx); err != nil {
		log.Printf("market cache unavailable: %v", err)
	} else {
		cache = client
		defer client.Close()
	}

	var ohlcvStore *ohlcv.TimescaleStore
	var spreadStore *engticker.SpreadStore
	if timescalePool, err := storage.NewTimescalePoolFromEnv(ctx); err != nil {
		log.Printf("market timescale unavailable: %v", err)
	} else {
		defer timescalePool.Close()
		ohlcvStore = ohlcv.NewTimescaleStore(timescalePool)
		spreadStore = engticker.NewSpreadStore(timescalePool)
	}

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
	authMiddleware := middleware.NewAuthMiddleware(authService)

	service := marketapi.NewService(postgresPool, ohlcvStore, spreadStore, cache)
	marketHandler := handler.NewMarketHandler(service, authMiddleware)
	defiService := defiapi.NewService(postgresPool)
	defiHandler := handler.NewDefiHandler(defiService)
	newsService := newsapi.NewService(postgresPool)
	newsHandler := handler.NewNewsHandler(newsService)
	downloadCfg, err := irag.ConfigFromEnv()
	if err == nil {
		downloadService := downloadapi.NewService(downloadCfg, nil, nil)
		if downloadService.Enabled() {
			log.Printf("download irag service initialized")
		} else {
			log.Printf("download irag unavailable")
		}
	} else {
		log.Printf("download irag unavailable: %v", err)
	}
	quantService := quantapi.NewService(postgresPool)
	quantHandler := handler.NewQuantHandler(quantService)
	router := api.NewRouter(marketHandler, defiHandler, newsHandler, authHandler, quantHandler)

	port := resolvePort(os.Getenv("API_PORT"))
	server := newHTTPServer(port, router)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	go func() {
		if err := bootstrapMarketCatalog(ctx, postgresPool); err != nil {
			log.Printf("market bootstrap skipped: %v", err)
		}
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

func bootstrapMarketCatalog(ctx context.Context, postgresPool *pgxpool.Pool) error {
	if postgresPool == nil {
		return fmt.Errorf("postgres pool is required")
	}

	store := coingecko.NewStore(postgresPool)
	count, err := store.CountCoins(ctx)
	if err != nil {
		return err
	}
	var pricedCount int
	if err := postgresPool.QueryRow(ctx, `SELECT count(*) FROM cold_coin_data WHERE current_price_usd IS NOT NULL`).Scan(&pricedCount); err != nil {
		return err
	}
	if count > 0 && pricedCount > 0 {
		return nil
	}

	fetcher, err := coingecko.NewFetcherFromEnv()
	if err != nil {
		return err
	}

	service := coingecko.NewService(fetcher, store, 4, 250)
	bootstrapCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if _, err := service.RunOnce(bootstrapCtx); err != nil {
		return err
	}

	log.Printf("market catalog bootstrapped from coingecko source")
	return nil
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
