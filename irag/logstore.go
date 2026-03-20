package irag

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LogStore struct {
	pool *pgxpool.Pool
}

func NewLogStore(pool *pgxpool.Pool) *LogStore {
	if pool == nil {
		return nil
	}
	return &LogStore{pool: pool}
}

type RequestLog struct {
	Endpoint          string
	Category          string
	ProviderUsed      string
	FallbackChain     []string
	Status            string
	HTTPStatus        int
	LatencyMS         int
	ResponseSizeBytes int
	CacheKey          string
	CacheTTLSeconds   int
	ErrorCode         string
	ErrorMessage      string
	ClientID          string
	IsPremium         bool
}

func (s *LogStore) Insert(ctx context.Context, log RequestLog) {
	if s == nil || s.pool == nil {
		return
	}

	_, _ = s.pool.Exec(ctx, `
		INSERT INTO irag_request_log (
			endpoint, category, provider_used, fallback_chain, status, http_status,
			latency_ms, response_size_bytes, cache_key, cache_ttl_seconds, error_code,
			error_message, client_id, is_premium
		) VALUES (
			$1, $2, $3, string_to_array(NULLIF($4, ''), ','), $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14
		)`,
		log.Endpoint, log.Category, nullableString(log.ProviderUsed), strings.Join(log.FallbackChain, ","), log.Status, nullableInt(log.HTTPStatus),
		nullableInt(log.LatencyMS), nullableInt(log.ResponseSizeBytes), nullableString(log.CacheKey), nullableInt(log.CacheTTLSeconds), nullableString(log.ErrorCode),
		nullableString(log.ErrorMessage), nullableString(log.ClientID), log.IsPremium,
	)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}
