package irag

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type ProviderName string

const (
	ProviderKanata    ProviderName = "kanata"
	ProviderNexure    ProviderName = "nexure"
	ProviderRyzumi    ProviderName = "ryzumi"
	ProviderChocomilk ProviderName = "chocomilk"
	ProviderYTDLP     ProviderName = "ytdlp"
)

type UpstreamConfig struct {
	Name       ProviderName
	BaseURL    *url.URL
	HostHeader string
	Enabled    bool
}

type Config struct {
	Timeout         time.Duration
	DefaultCacheTTL time.Duration
	CacheEnabled    bool
	LogEnabled      bool
	AllowedOrigins  []string
	Upstreams       map[ProviderName]UpstreamConfig
}

func ConfigFromEnv() (Config, error) {
	cfg := Config{
		Timeout:         parseDurationEnv("IRAG_TIMEOUT_MS", 15*time.Second),
		DefaultCacheTTL: parseDurationEnv("IRAG_DEFAULT_CACHE_TTL", 5*time.Minute),
		CacheEnabled:    parseBoolEnv("IRAG_CACHE_ENABLED", true),
		LogEnabled:      parseBoolEnv("IRAG_LOG_ENABLED", true),
		AllowedOrigins:  parseCSVEnv("IRAG_ALLOWED_ORIGINS"),
		Upstreams:       make(map[ProviderName]UpstreamConfig),
	}

	upstreams := []struct {
		name       ProviderName
		urlEnv     string
		defaultURL string
		hostHeader string
	}{
		{ProviderKanata, "IRAG_KANATA_URL", "https://api.kanata.web.id", ""},
		{ProviderNexure, "IRAG_NEXURE_URL", "https://api.ammaricano.my.id", ""},
		{ProviderRyzumi, "IRAG_RYZUMI_URL", "https://api.ryzumi.net", ""},
		{ProviderChocomilk, "IRAG_CHOCOMILK_URL", "https://chocomilk.amira.us.kg", ""},
		{ProviderYTDLP, "IRAG_YTDLP_URL", "https://ytdlpyton.nvlgroup.my.id", ""},
	}

	for _, upstream := range upstreams {
		base := strings.TrimSpace(os.Getenv(upstream.urlEnv))
		if base == "" {
			base = upstream.defaultURL
		}
		if base == "" {
			continue
		}

		parsed, err := url.Parse(base)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", upstream.urlEnv, err)
		}

		cfg.Upstreams[upstream.name] = UpstreamConfig{
			Name:       upstream.name,
			BaseURL:    parsed,
			HostHeader: upstream.hostHeader,
			Enabled:    true,
		}
	}

	return cfg, nil
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if strings.HasSuffix(raw, "ms") || strings.Contains(raw, "s") || strings.Contains(raw, "m") || strings.Contains(raw, "h") {
		if value, err := time.ParseDuration(raw); err == nil && value > 0 {
			return value
		}
	}
	if millis, err := strconv.Atoi(raw); err == nil && millis > 0 {
		return time.Duration(millis) * time.Millisecond
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
