package irag

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultUpstreamUserAgent = "Mozilla/5.0 (dwizzyBRAIN IRAG; +https://dwizzy.my.id)"

type Service struct {
	cfg     Config
	cache   Cache
	logs    *LogStore
	client  *http.Client
	circuit *CircuitBreaker
}

func NewService(cfg Config, cache Cache, logs *LogStore) *Service {
	return &Service{
		cfg:     cfg,
		cache:   cache,
		logs:    logs,
		client:  &http.Client{},
		circuit: NewCircuitBreaker(3, 60*time.Second),
	}
}

func (s *Service) Enabled() bool {
	return len(s.cfg.Upstreams) > 0
}

type ProviderInfo struct {
	ID      string `json:"id"`
	BaseURL string `json:"base_url"`
	Status  string `json:"status"`
}

func (s *Service) ProviderSnapshot() []ProviderInfo {
	infos := make([]ProviderInfo, 0, len(s.cfg.Upstreams))
	for _, provider := range []ProviderName{ProviderKanata, ProviderNexure, ProviderRyzumi, ProviderChocomilk, ProviderYTDLP} {
		upstream, ok := s.cfg.Upstreams[provider]
		if !ok || upstream.BaseURL == nil {
			continue
		}
		infos = append(infos, ProviderInfo{
			ID:      string(provider),
			BaseURL: upstream.BaseURL.String(),
			Status:  "configured",
		})
	}
	return infos
}

func (s *Service) ProviderDetail(id string) (ProviderInfo, bool) {
	for _, provider := range s.ProviderSnapshot() {
		if strings.EqualFold(provider.ID, id) {
			return provider, true
		}
	}
	return ProviderInfo{}, false
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.writeCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if s.handleToBase64(w, r) {
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/v1/") {
		writeEnvelopeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]any{"message": "not found"},
		})
		return
	}

	spec := s.routeSpecForPath(r.URL.Path)
	if len(spec.Providers) == 0 {
		writeEnvelopeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]any{"message": "route not found"},
		})
		return
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	cacheKey := s.cacheKey(r.Method, r.URL, body)
	if spec.CacheTTL > 0 && s.cfg.CacheEnabled && s.cache != nil && r.Method == http.MethodGet {
		if cached, ok, err := s.cache.Get(r.Context(), cacheKey); err == nil && ok {
			s.writeCached(w, r, cached, spec.CacheTTL)
			s.log(r, spec, cached.Provider, []string{}, "cache_hit_l1", cached.Status, 0, len(cached.Body), cacheKey, spec.CacheTTL, "", "", true)
			return
		}
	}

	start := time.Now()
	resp, attempted, err := s.proxyWithFallback(r.Context(), r, spec, body)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		s.writeFailure(w, status, spec, attempted, err)
		s.log(r, spec, attemptedProvider(attempted), attempted, "provider_error", status, time.Since(start), 0, cacheKey, spec.CacheTTL, "provider_error", err.Error(), false)
		return
	}

	if spec.CacheTTL > 0 && s.cfg.CacheEnabled && s.cache != nil && r.Method == http.MethodGet {
		_ = s.cache.Set(r.Context(), cacheKey, resp.cachedResponse(), spec.CacheTTL)
	}

	s.writeResponse(w, r, resp)
	s.log(r, spec, resp.Provider, attempted, resp.StatusClass(), resp.Status, resp.Latency, len(resp.Body), cacheKey, spec.CacheTTL, "", "", false)
}

type routeSpec struct {
	Category  string
	Providers []ProviderName
	CacheTTL  time.Duration
	Timeout   time.Duration
}

type proxyResponse struct {
	Status        int
	Headers       http.Header
	Body          []byte
	ContentType   string
	Raw           bool
	Provider      string
	FallbackChain []string
	Latency       time.Duration
}

func (p proxyResponse) cachedResponse() CachedResponse {
	return CachedResponse{
		Status:      p.Status,
		ContentType: p.ContentType,
		Body:        p.Body,
		Raw:         p.Raw,
		Provider:    p.Provider,
	}
}

func (p proxyResponse) StatusClass() string {
	switch {
	case p.Status >= 200 && p.Status < 300 && len(p.FallbackChain) > 1:
		return "fallback_used"
	case p.Status >= 200 && p.Status < 300:
		return "success"
	case p.Status == http.StatusGatewayTimeout:
		return "timeout"
	case p.Status == http.StatusTooManyRequests:
		return "provider_error"
	default:
		return "provider_error"
	}
}

func (s *Service) proxyWithFallback(ctx context.Context, r *http.Request, spec routeSpec, body []byte) (proxyResponse, []string, error) {
	attempted := make([]string, 0, len(spec.Providers))
	for _, provider := range spec.Providers {
		if !s.circuit.Allow(provider) {
			attempted = append(attempted, string(provider))
			continue
		}
		resp, err := s.attemptProvider(ctx, r, provider, body, spec.Timeout)
		attempted = append(attempted, string(provider))
		if err != nil {
			s.circuit.Failure(provider)
			continue
		}
		if resp.Status >= 200 && resp.Status < 300 {
			s.circuit.Success(provider)
			resp.FallbackChain = append([]string(nil), attempted...)
			return resp, attempted, nil
		}
		if shouldRetryStatusForRoute(r.URL.Path, resp.Status) {
			s.circuit.Failure(provider)
			continue
		}
		s.circuit.Success(provider)
		resp.FallbackChain = append([]string(nil), attempted...)
		return resp, attempted, nil
	}
	return proxyResponse{}, attempted, fmt.Errorf("all providers failed for %s", r.URL.Path)
}

func (s *Service) attemptProvider(ctx context.Context, r *http.Request, provider ProviderName, body []byte, timeout time.Duration) (proxyResponse, error) {
	upstream, ok := s.cfg.Upstreams[provider]
	if !ok || !upstream.Enabled || upstream.BaseURL == nil {
		return proxyResponse{}, fmt.Errorf("provider %s unavailable", provider)
	}

	reqCtx := ctx
	var cancel context.CancelFunc
	if timeout <= 0 {
		timeout = s.cfg.Timeout
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	reqCtx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	upstreamURL := cloneURL(upstream.BaseURL)
	proxyPath, proxyQuery := s.upstreamPathAndQuery(r.URL.Path, provider, r.URL.Query())
	upstreamMethod := r.Method
	if isUtilityUpscaleRoute(r.URL.Path) {
		if formValues, ok := parseURLEncodedBody(body); ok {
			for key, values := range formValues {
				for _, value := range values {
					proxyQuery.Add(key, value)
				}
			}
		}
		if r.Method != http.MethodGet {
			upstreamMethod = http.MethodGet
		}
	}
	upstreamURL.Path = joinURLPath(upstreamURL.Path, proxyPath)
	upstreamURL.RawQuery = proxyQuery.Encode()

	reqBody := io.NopCloser(bytes.NewReader(body))
	if upstreamMethod == http.MethodGet {
		reqBody = http.NoBody
	}
	if len(body) == 0 {
		reqBody = http.NoBody
	}

	req, err := http.NewRequestWithContext(reqCtx, upstreamMethod, upstreamURL.String(), reqBody)
	if err != nil {
		return proxyResponse{}, err
	}
	copyProxyHeaders(req.Header, r.Header)
	setDefaultUpstreamHeaders(req.Header)
	req.Header.Del("Host")
	req.Header.Del("Content-Length")
	req.Header.Del("Accept-Encoding")
	req.Header.Del("Connection")
	if upstream.HostHeader != "" {
		req.Host = upstream.HostHeader
	} else {
		req.Host = upstream.BaseURL.Host
	}

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		return proxyResponse{}, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return proxyResponse{}, err
	}

	contentType := resp.Header.Get("Content-Type")
	raw := shouldPassThroughRaw(r.URL.Path, contentType)
	out := proxyResponse{
		Status:      resp.StatusCode,
		Headers:     resp.Header.Clone(),
		Body:        payload,
		ContentType: contentType,
		Raw:         raw,
		Provider:    string(provider),
		Latency:     time.Since(start),
	}
	return out, nil
}

func isUtilityUpscaleRoute(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	return lower == "/v1/utility/upscale" || lower == "/v1/tools/upscale"
}

func parseURLEncodedBody(body []byte) (url.Values, bool) {
	if len(body) == 0 {
		return nil, false
	}
	values, err := url.ParseQuery(string(body))
	if err != nil || len(values) == 0 {
		return nil, false
	}
	return values, true
}

func setDefaultUpstreamHeaders(header http.Header) {
	if ua := strings.TrimSpace(header.Get("User-Agent")); ua == "" || strings.EqualFold(ua, "curl/7.64.0") || strings.EqualFold(ua, "curl/8.0.1") || strings.HasPrefix(strings.ToLower(ua), "curl/") {
		header.Set("User-Agent", defaultUpstreamUserAgent)
	}
	if header.Get("Accept") == "" {
		header.Set("Accept", "application/json,text/plain,text/html;q=0.9,*/*;q=0.8")
	}
	if header.Get("Accept-Language") == "" {
		header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	}
	if header.Get("Referer") == "" {
		header.Set("Referer", "https://www.google.com/")
	}
}

func (s *Service) upstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasPrefix(lower, "/v1/ai/text/"):
		slug := strings.TrimPrefix(lower, "/v1/ai/text/")
		switch provider {
		case ProviderNexure:
			return nexureAIPathForSlug(slug), buildNexureAIQueryForSlug(slug, query)
		case ProviderRyzumi:
			return ryzumiAIPathForSlug(slug), buildRyzumiAIQueryForSlug(slug, query)
		case ProviderChocomilk:
			return chocomilkAIPathForSlug(slug), buildChocomilkAIQuery(query)
		case ProviderYTDLP:
			return ytdlpAIPathForSlug(slug), buildYTDLPAIQuery(slug, query)
		default:
			return path, cloneValues(query)
		}
	case lower == "/v1/ai/generate":
		switch provider {
		case ProviderKanata:
			return "/ai/generate", buildAIImageQuery(query)
		default:
			return path, cloneValues(query)
		}
	case lower == "/v1/ai/image":
		switch provider {
		case ProviderKanata:
			return "/ai/image", buildAIImageQuery(query)
		default:
			return path, cloneValues(query)
		}
	case strings.HasPrefix(lower, "/v1/ai/image/"):
		slug := strings.TrimPrefix(lower, "/v1/ai/image/")
		switch provider {
		case ProviderNexure, ProviderRyzumi, ProviderChocomilk:
			return "/api/ai/" + slug, buildAIImageQuery(query)
		case ProviderKanata:
			return "/ai/image", buildAIImageQuery(query)
		default:
			return path, cloneValues(query)
		}
	case strings.HasPrefix(lower, "/v1/ai/process/"), strings.HasPrefix(lower, "/v1/i2i/"):
		slug := strings.TrimPrefix(strings.TrimPrefix(lower, "/v1/ai/process/"), "/v1/i2i/")
		return aiProcessUpstreamPathAndQuery(slug, provider, query)
	case strings.HasPrefix(lower, "/v1/ai/"):
		slug := strings.TrimPrefix(lower, "/v1/ai/")
		if slug == "" {
			return path, cloneValues(query)
		}
		if isDirectAIImageSlug(slug) {
			return aiDirectUpstreamPathAndQuery(slug, provider, query)
		}
		return aiDirectUpstreamPathAndQuery(slug, provider, query)
	case strings.HasPrefix(lower, "/v1/youtube/"):
		return youtubeUpstreamPathAndQuery(lower, provider, query)
	case lower == "/v1/llm/chatgpt/completions":
		return llmUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/download/"):
		return s.downloadUpstreamPathAndQuery(path, provider, query)
	case strings.HasPrefix(lower, "/v1/search/"):
		return s.searchUpstreamPathAndQuery(path, provider, query)
	case strings.HasPrefix(lower, "/v1/stalk/"):
		return s.stalkUpstreamPathAndQuery(path, provider, query)
	case strings.HasPrefix(lower, "/v1/bmkg/"):
		if provider == ProviderKanata {
			return bmkgUpstreamPathAndQuery(lower, query)
		}
		return path, cloneValues(query)
	case strings.HasPrefix(lower, "/v1/anime/"):
		return animeUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/manga/"):
		return mangaUpstreamPathAndQuery(lower, provider, query)
	case lower == "/v1/otakudesu" || strings.HasPrefix(lower, "/v1/otakudesu/"):
		return animeUpstreamPathAndQuery(otakudesuAliasPath(lower), provider, query)
	case lower == "/v1/komiku" || strings.HasPrefix(lower, "/v1/komiku/"):
		return mangaUpstreamPathAndQuery(komikuAliasPath(lower), provider, query)
	case strings.HasPrefix(lower, "/v1/film/"), strings.HasPrefix(lower, "/v1/drama/"), strings.HasPrefix(lower, "/v1/lk21"):
		return filmUpstreamPathAndQuery(lower, provider, query)
	case lower == "/v1/novel" || strings.HasPrefix(lower, "/v1/novel/"):
		return novelUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/news/"):
		return newsUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/weebs/"):
		return weebsUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/misc/"):
		return miscUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/media/"):
		return mediaUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/upload/"):
		return uploadUpstreamPathAndQuery(lower, provider, query)
	case lower == "/v1/upload":
		return uploadUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/dramabox"):
		return dramaboxUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/tools/cctv"), strings.HasPrefix(lower, "/v1/bsw/cctv"):
		return cctvUpstreamPathAndQuery(lower, provider, query)
	case lower == "/v1/utility/upscale" || lower == "/v1/tools/upscale":
		return s.toolsUpstreamPathAndQuery(lower, provider, query)
	case lower == "/v1/misc/server-info" || lower == "/v1/server-info":
		return serverInfoUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/game/"):
		return gameUpstreamPathAndQuery(lower, provider, query)
	case strings.HasPrefix(lower, "/v1/tools/"):
		return s.toolsUpstreamPathAndQuery(path, provider, query)
	case strings.HasPrefix(lower, "/v1/islamic/"):
		if provider == ProviderYTDLP {
			return ytdlpIslamicPathAndQuery(lower, query)
		}
		return path, cloneValues(query)
	default:
		return path, cloneValues(query)
	}
}

func buildAIImageQuery(query url.Values) url.Values {
	out := url.Values{}
	if prompt := strings.TrimSpace(query.Get("prompt")); prompt != "" {
		out.Set("prompt", prompt)
	} else if prompt := aiPromptValue(query); prompt != "" {
		out.Set("prompt", prompt)
	}
	if model := strings.TrimSpace(query.Get("model")); model != "" {
		out.Set("model", model)
	}
	if style := strings.TrimSpace(query.Get("style")); style != "" {
		out.Set("style", style)
	}
	if size := strings.TrimSpace(query.Get("size")); size != "" {
		out.Set("size", size)
	}
	if session := strings.TrimSpace(query.Get("session")); session != "" {
		out.Set("session", session)
	}
	return out
}

func (s *Service) searchUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	lower := strings.ToLower(path)
	slug := strings.TrimPrefix(lower, "/v1/search/")
	out := buildSearchQuery(query)

	switch provider {
	case ProviderRyzumi:
		switch {
		case strings.HasPrefix(slug, "google/image"):
			return "/api/search/gimage", out
		case strings.HasPrefix(slug, "google"):
			return "/api/search/google", out
		case strings.HasPrefix(slug, "bilibili"):
			return "/api/search/bilibili", out
		case strings.HasPrefix(slug, "bmkg"):
			return "/api/search/bmkg", out
		case strings.HasPrefix(slug, "chord"):
			return "/api/search/chord", out
		case strings.HasPrefix(slug, "harga-emas"), strings.HasPrefix(slug, "gold"):
			return "/api/search/harga-emas", out
		case strings.HasPrefix(slug, "jadwal-sholat"), strings.HasPrefix(slug, "sholat"):
			if city := strings.TrimSpace(firstNonEmpty(query.Get("kota"), query.Get("city"), query.Get("q"), query.Get("query"))); city != "" {
				out = url.Values{}
				out.Set("kota", city)
				out.Set("city", city)
			}
			return "/api/search/jadwal-sholat", out
		case strings.HasPrefix(slug, "kurs-bca"):
			return "/api/search/kurs-bca", out
		case strings.HasPrefix(slug, "lens"):
			if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"), query.Get("imgUrl"))); u != "" {
				out = url.Values{}
				out.Set("url", u)
			}
			return "/api/search/lens", out
		case strings.HasPrefix(slug, "youtube"):
			return "/api/search/yt", out
		case strings.HasPrefix(slug, "spotify"):
			return "/api/search/spotify", out
		case strings.HasPrefix(slug, "pinterest"):
			return "/api/search/pinterest", out
		case strings.HasPrefix(slug, "pixiv"):
			if q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("query", q)
				out.Set("q", q)
			}
			return "/api/search/pixiv", out
		case strings.HasPrefix(slug, "lyrics"):
			return "/api/search/lyrics", out
		case strings.HasPrefix(slug, "mahasiswa"), strings.HasPrefix(slug, "pddikti"):
			if q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("name"))); q != "" {
				out = url.Values{}
				out.Set("query", q)
				out.Set("q", q)
			}
			return "/api/search/mahasiswa", out
		case strings.HasPrefix(slug, "wallpaper"):
			if q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("query", q)
				out.Set("q", q)
			}
			return "/api/search/wallpaper-moe", out
		case strings.HasPrefix(slug, "tiktok"):
			return "/api/search/tiktok", out
		case strings.HasPrefix(slug, "tidal"):
			return "/api/search/tidal", out
		case strings.HasPrefix(slug, "anime"):
			if q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("query", q)
				out.Set("q", q)
			}
			return "/api/search/anime", out
		case strings.HasPrefix(slug, "manga"):
			if q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("query", q)
				out.Set("q", q)
			}
			return "/api/search/manga", out
		case strings.HasPrefix(slug, "weather"):
			if city := strings.TrimSpace(firstNonEmpty(query.Get("city"), query.Get("q"), query.Get("query"))); city != "" {
				out = url.Values{}
				out.Set("city", city)
			}
			return "/api/search/weather", out
		case strings.HasPrefix(slug, "film"), strings.HasPrefix(slug, "drama"):
			if q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("query", q)
				out.Set("q", q)
			}
			return "/api/dramabox/search", out
		case strings.HasPrefix(slug, "bstation"), strings.HasPrefix(slug, "bilibili"):
			if q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("query", q)
				out.Set("q", q)
			}
			return "/api/search/bstation", out
		case strings.HasPrefix(slug, "novel"):
			return "/api/search/novel", out
		default:
			return "/api/search/" + slug, out
		}
	case ProviderNexure:
		switch {
		case strings.HasPrefix(slug, "google/image"):
			return "/api/search/gimage", out
		case strings.HasPrefix(slug, "google"):
			return "/api/search/google", out
		case strings.HasPrefix(slug, "youtube"):
			return "/api/search/youtube", out
		case strings.HasPrefix(slug, "spotify"):
			return "/api/search/spotify", out
		case strings.HasPrefix(slug, "pinterest"):
			return "/api/search/pinterest", out
		case strings.HasPrefix(slug, "lyrics"):
			return "/api/search/lyrics", out
		case strings.HasPrefix(slug, "anime"):
			return "/api/otakudesu/search", out
		case strings.HasPrefix(slug, "manga"):
			return "/api/komiku/search", out
		case strings.HasPrefix(slug, "novel"):
			return "/api/search/novel", out
		case strings.HasPrefix(slug, "bstation"), strings.HasPrefix(slug, "bilibili"):
			return "/api/search/bstation", out
		case strings.HasPrefix(slug, "cookpad"):
			return "/api/search/cookpad", out
		case strings.HasPrefix(slug, "wallpaper"):
			return "/api/search/minwall", out
		case strings.HasPrefix(slug, "pddikti"), strings.HasPrefix(slug, "mahasiswa"):
			return "/api/search/pddikti", out
		case strings.HasPrefix(slug, "film"), strings.HasPrefix(slug, "drama"):
			return "/api/dramabox/search", out
		case strings.HasPrefix(slug, "tiktok"):
			return "/api/search/tiktok", out
		case strings.HasPrefix(slug, "tidal"):
			return "/api/search/tidal", out
		default:
			return "/api/search/" + slug, out
		}
	case ProviderChocomilk:
		return "/search/", out
	case ProviderKanata:
		switch {
		case strings.HasPrefix(slug, "anime"):
			return "/otakudesu/search", out
		case strings.HasPrefix(slug, "manga"):
			return "/komiku/search", out
		case strings.HasPrefix(slug, "film"):
			return "/nontonfilm/search", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func (s *Service) stalkUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	lower := strings.ToLower(path)
	slug := strings.TrimPrefix(lower, "/v1/stalk/")
	out := buildStalkQuery(slug, query)

	switch provider {
	case ProviderNexure:
		switch {
		case strings.Contains(slug, "instagram"):
			return "/api/stalk/instagram", out
		default:
			return "/api/stalk/" + slug, out
		}
	case ProviderRyzumi:
		switch {
		case strings.Contains(slug, "github"):
			return "/api/stalk/github", out
		case strings.Contains(slug, "mobile-legends"), slug == "ml":
			return "/api/stalk/mobile-legends", out
		case strings.Contains(slug, "free-fire"), slug == "freefire":
			return "/api/stalk/free-fire", out
		case strings.Contains(slug, "valorant"):
			return "/api/stalk/valorant", out
		case strings.Contains(slug, "clash-of-clans"):
			return "/api/stalk/clash-of-clans", out
		case strings.Contains(slug, "clash-royale"):
			return "/api/stalk/clash-royale", out
		case strings.Contains(slug, "npm"):
			return "/api/stalk/npm", out
		case strings.Contains(slug, "tiktok"):
			return "/api/stalk/tiktok", out
		case strings.Contains(slug, "twitter"):
			return "/api/stalk/twitter", out
		case strings.Contains(slug, "youtube"):
			return "/api/stalk/youtube", out
		default:
			return "/api/stalk/" + slug, out
		}
	default:
		return path, cloneValues(query)
	}
}

func buildSearchQuery(query url.Values) url.Values {
	out := cloneValues(query)
	if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("ask"), query.Get("text"))); q != "" {
		out.Set("q", q)
		out.Set("query", q)
	}
	if page := strings.TrimSpace(query.Get("page")); page != "" {
		out.Set("page", page)
	}
	if limit := strings.TrimSpace(query.Get("limit")); limit != "" {
		out.Set("limit", limit)
	}
	if typ := strings.TrimSpace(query.Get("type")); typ != "" {
		out.Set("type", typ)
	}
	return out
}

func buildStalkQuery(slug string, query url.Values) url.Values {
	out := cloneValues(query)
	switch {
	case strings.Contains(slug, "instagram"), strings.Contains(slug, "github"), strings.Contains(slug, "npm"):
		if username := strings.TrimSpace(firstNonEmpty(query.Get("username"), query.Get("user"), query.Get("q"), query.Get("query"))); username != "" {
			out.Set("username", username)
		}
	case strings.Contains(slug, "mobile-legends"), slug == "ml", strings.Contains(slug, "free-fire"), slug == "freefire":
		if id := strings.TrimSpace(firstNonEmpty(query.Get("id"), query.Get("userId"), query.Get("user_id"), query.Get("player_id"), query.Get("uid"))); id != "" {
			out.Set("id", id)
			out.Set("userId", id)
		}
		if server := strings.TrimSpace(firstNonEmpty(query.Get("server"), query.Get("zoneId"), query.Get("zone_id"), query.Get("zone"))); server != "" {
			out.Set("server", server)
			out.Set("zoneId", server)
		}
	case strings.Contains(slug, "valorant"):
		if name := strings.TrimSpace(firstNonEmpty(query.Get("name"), query.Get("username"), query.Get("q"), query.Get("query"))); name != "" {
			out.Set("name", name)
		}
		if tag := strings.TrimSpace(firstNonEmpty(query.Get("tag"), query.Get("server"), query.Get("zone"))); tag != "" {
			out.Set("tag", tag)
		}
	case strings.Contains(slug, "clash-of-clans"), strings.Contains(slug, "clash-royale"):
		if name := strings.TrimSpace(firstNonEmpty(query.Get("name"), query.Get("username"), query.Get("q"), query.Get("query"))); name != "" {
			out.Set("name", name)
		}
		if token := strings.TrimSpace(firstNonEmpty(query.Get("token"), query.Get("apikey"), query.Get("api_key"))); token != "" {
			out.Set("token", token)
		}
	}
	return out
}

func nexureAIPathForSlug(slug string) string {
	switch slug {
	case "gpt":
		return "/api/ai/gpt"
	case "gpt-v2":
		return "/api/ai/v2/gpt"
	case "claila":
		return "/api/ai/claila"
	case "copilot":
		return "/api/ai/copilot"
	case "gemini":
		return "/api/ai/gemini"
	case "deepseek":
		return "/api/ai/deepseek"
	case "groq":
		return "/api/ai/groq"
	case "meta":
		return "/api/ai/meta"
	case "perplexity":
		return "/api/ai/perplexity"
	case "pollinations":
		return "/api/ai/pollinations"
	case "qwen":
		return "/api/ai/qwen"
	case "webpilot":
		return "/api/ai/webpilot"
	case "ai4chat":
		return "/api/ai/ai4chat"
	case "z-ai":
		return "/api/ai/z-ai"
	default:
		return "/api/ai/gpt"
	}
}

func ryzumiAIPathForSlug(slug string) string {
	switch slug {
	case "chatgpt-ryz", "gpt":
		return "/api/ai/chatgpt"
	case "deepseek-ryz", "deepseek":
		return "/api/ai/deepseek"
	case "gemini-ryz", "gemini":
		return "/api/ai/gemini"
	case "mistral":
		return "/api/ai/mistral"
	case "qwen":
		return "/api/ai/qwen"
	default:
		return "/api/ai/chatgpt"
	}
}

func chocomilkAIPathForSlug(slug string) string {
	switch slug {
	case "chocomilk-gpt":
		return "/api/ai/chatgpt"
	default:
		return "/api/ai/chatgpt"
	}
}

func ytdlpAIPathForSlug(slug string) string {
	switch slug {
	case "gemini":
		return "/ai/gemini"
	case "powerbrain":
		return "/ai/powerbrain"
	case "felo":
		return "/ai/felo"
	case "beago":
		return "/ai/beago"
	case "deepai-chat":
		return "/ai/deepai-chat"
	default:
		return "/ai/gemini"
	}
}

func buildNexureAIQuery(query url.Values) url.Values {
	out := url.Values{}
	if ask := aiPromptValue(query); ask != "" {
		out.Set("ask", ask)
	}
	if model := strings.TrimSpace(query.Get("model")); model != "" {
		out.Set("model", model)
	}
	if style := strings.TrimSpace(query.Get("style")); style != "" {
		out.Set("style", style)
	}
	if session := strings.TrimSpace(query.Get("session")); session != "" {
		out.Set("session", session)
	}
	if imageURL := strings.TrimSpace(query.Get("imageUrl")); imageURL != "" {
		out.Set("imageUrl", imageURL)
	}
	if temperature := strings.TrimSpace(query.Get("temperature")); temperature != "" {
		out.Set("temperature", temperature)
	}
	if think := strings.TrimSpace(query.Get("think")); think != "" {
		out.Set("think", think)
	}
	return out
}

func buildNexureAIQueryForSlug(slug string, query url.Values) url.Values {
	out := buildNexureAIQuery(query)
	switch slug {
	case "groq":
		if out.Get("model") == "" {
			out.Set("model", "groq/compound")
		}
	case "qwen":
		if out.Get("model") == "" {
			out.Set("model", "qwen3-coder-plus")
		}
	case "claila":
		if out.Get("model") == "" {
			out.Set("model", "gpt-5-mini")
		}
	}
	return out
}

func buildRyzumiAIQuery(query url.Values) url.Values {
	out := url.Values{}
	if text := aiPromptValue(query); text != "" {
		out.Set("text", text)
		out.Set("prompt", text)
	}
	if model := strings.TrimSpace(query.Get("model")); model != "" {
		out.Set("model", model)
	}
	if session := strings.TrimSpace(query.Get("session")); session != "" {
		out.Set("session", session)
	}
	if imageURL := strings.TrimSpace(query.Get("imageUrl")); imageURL != "" {
		out.Set("imageUrl", imageURL)
	}
	if style := strings.TrimSpace(query.Get("style")); style != "" {
		out.Set("style", style)
	}
	return out
}

func buildRyzumiAIQueryForSlug(slug string, query url.Values) url.Values {
	out := buildRyzumiAIQuery(query)
	switch slug {
	case "qwen":
		if out.Get("model") == "" {
			out.Set("model", "qwen3-coder-plus")
		}
	}
	return out
}

func buildChocomilkAIQuery(query url.Values) url.Values {
	out := url.Values{}
	if ask := aiPromptValue(query); ask != "" {
		out.Set("ask", ask)
		out.Set("prompt", ask)
		out.Set("text", ask)
	}
	if model := strings.TrimSpace(query.Get("model")); model != "" {
		out.Set("model", model)
	}
	if session := strings.TrimSpace(query.Get("session")); session != "" {
		out.Set("session", session)
	}
	return out
}

func llmChain(path string) []ProviderName {
	return []ProviderName{ProviderChocomilk, ProviderNexure, ProviderRyzumi}
}

func llmUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	switch provider {
	case ProviderChocomilk:
		return "/api/ai/chatgpt", buildChocomilkAIQuery(query)
	case ProviderNexure:
		return "/api/ai/gpt", buildNexureAIQueryForSlug("gpt", query)
	case ProviderRyzumi:
		return "/api/ai/chatgpt", buildRyzumiAIQueryForSlug("gpt", query)
	default:
		return path, cloneValues(query)
	}
}

func isDirectAIImageSlug(slug string) bool {
	return strings.Contains(slug, "animagine-xl-") || strings.Contains(slug, "deepimg") || strings.Contains(slug, "flux-schnell") || strings.Contains(slug, "pollinations/image")
}

func isAllowedDirectAITextSlug(slug string) bool {
	switch strings.TrimSpace(strings.ToLower(slug)) {
	case "ai4chat",
		"beago",
		"chatgpt-ryz",
		"chocomilk-gpt",
		"claila",
		"copilot",
		"deepai-chat",
		"deepseek",
		"deepseek-ryz",
		"felo",
		"gemini",
		"gemini-ryz",
		"gpt",
		"gpt-v2",
		"groq",
		"meta",
		"mistral",
		"perplexity",
		"powerbrain",
		"qwen",
		"v2/gpt",
		"webpilot",
		"z-ai":
		return true
	default:
		return false
	}
}

func isAllowedYouTubeSlug(slug string) bool {
	switch strings.TrimSpace(strings.ToLower(slug)) {
	case "download", "info", "play", "search":
		return true
	default:
		return false
	}
}

func aiDirectTextChain(path string) []ProviderName {
	return aiTextChain("/v1/ai/text/" + strings.TrimPrefix(strings.ToLower(path), "/v1/ai/"))
}

func aiDirectImageChain(path string) []ProviderName {
	slug := strings.TrimPrefix(strings.ToLower(path), "/v1/ai/")
	if strings.Contains(slug, "kanata") {
		return []ProviderName{ProviderKanata, ProviderNexure, ProviderRyzumi, ProviderChocomilk}
	}
	return []ProviderName{ProviderKanata, ProviderNexure, ProviderRyzumi, ProviderChocomilk}
}

func aiDirectUpstreamPathAndQuery(slug string, provider ProviderName, query url.Values) (string, url.Values) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return "", cloneValues(query)
	}
	if isDirectAIImageSlug(slug) {
		switch provider {
		case ProviderKanata:
			return "/ai/image", buildAIImageQuery(query)
		case ProviderNexure, ProviderRyzumi, ProviderChocomilk:
			return "/api/ai/" + slug, buildAIImageQuery(query)
		default:
			return "/api/ai/" + slug, buildAIImageQuery(query)
		}
	}
	switch provider {
	case ProviderYTDLP:
		return ytdlpAIPathForSlug(slug), buildYTDLPAIQuery(slug, query)
	case ProviderNexure:
		return "/api/ai/" + slug, buildNexureAIQueryForSlug(strings.TrimPrefix(slug, "/"), query)
	case ProviderRyzumi:
		return ryzumiDirectAIPathForSlug(slug), buildRyzumiAIQueryForSlug(strings.TrimPrefix(slug, "/"), query)
	case ProviderChocomilk:
		return "/api/ai/chatgpt", buildChocomilkAIQuery(query)
	default:
		return "/api/ai/" + slug, cloneValues(query)
	}
}

func ryzumiDirectAIPathForSlug(slug string) string {
	switch slug {
	case "gpt", "v2/gpt", "ai4chat", "copilot", "claila", "meta", "perplexity", "webpilot", "z-ai":
		return "/api/ai/chatgpt"
	case "deepseek":
		return "/api/ai/deepseek"
	case "gemini":
		return "/api/ai/gemini"
	case "groq":
		return "/api/ai/groq"
	case "qwen":
		return "/api/ai/qwen"
	case "mistral":
		return "/api/ai/mistral"
	case "pollinations":
		return "/api/ai/pollinations"
	default:
		return "/api/ai/" + slug
	}
}

func buildDirectAIQuery(slug string, query url.Values) url.Values {
	if isDirectAIImageSlug(slug) {
		return buildAIImageQuery(query)
	}
	return buildNexureAIQueryForSlug(strings.TrimPrefix(slug, "/"), query)
}

func buildChocomilkYoutubeQuery(query url.Values) url.Values {
	out := url.Values{}
	if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("ask"), query.Get("text"), query.Get("keyword"))); q != "" {
		out.Set("q", q)
		out.Set("query", q)
		out.Set("search", q)
	}
	if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("link"), query.Get("source"), query.Get("target"))); u != "" {
		out.Set("url", u)
		out.Set("link", u)
	}
	if page := strings.TrimSpace(query.Get("page")); page != "" {
		out.Set("page", page)
	}
	if quality := strings.TrimSpace(firstNonEmpty(query.Get("quality"), query.Get("format"))); quality != "" {
		out.Set("quality", quality)
		out.Set("format", quality)
	}
	return out
}

func buildYTDLPYoutubeSearchQuery(query url.Values) url.Values {
	out := url.Values{}
	if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("ask"), query.Get("text"), query.Get("keyword"))); q != "" {
		out.Set("q", q)
		out.Set("query", q)
	}
	if page := strings.TrimSpace(query.Get("page")); page != "" {
		out.Set("page", page)
	}
	return out
}

func buildYTDLPYoutubeInfoQuery(query url.Values) url.Values {
	out := url.Values{}
	if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("link"), query.Get("source"), query.Get("target"))); u != "" {
		out.Set("url", u)
	}
	return out
}

func buildYTDLPYoutubeDownloadQuery(query url.Values) url.Values {
	out := buildYTDLPYoutubeInfoQuery(query)
	if quality := strings.TrimSpace(firstNonEmpty(query.Get("quality"), query.Get("format"), query.Get("itag"))); quality != "" {
		out.Set("quality", quality)
		out.Set("format", quality)
	}
	if lang := strings.TrimSpace(query.Get("lang")); lang != "" {
		out.Set("lang", lang)
	}
	return out
}

func buildNexureYoutubeSearchQuery(query url.Values) url.Values {
	out := url.Values{}
	if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("ask"), query.Get("text"), query.Get("keyword"))); q != "" {
		out.Set("q", q)
		out.Set("query", q)
	}
	if page := strings.TrimSpace(query.Get("page")); page != "" {
		out.Set("page", page)
	}
	return out
}

func youtubeChain(path string) []ProviderName {
	return []ProviderName{ProviderChocomilk, ProviderYTDLP, ProviderNexure}
}

func youtubeUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	lower := strings.ToLower(path)
	switch provider {
	case ProviderChocomilk:
		switch {
		case strings.HasSuffix(lower, "/search"):
			return "/v1/youtube/search", buildChocomilkYoutubeQuery(query)
		case strings.HasSuffix(lower, "/play"):
			return "/v1/youtube/play", buildChocomilkYoutubeQuery(query)
		case strings.HasSuffix(lower, "/info"):
			return "/v1/youtube/info", buildChocomilkYoutubeQuery(query)
		case strings.HasSuffix(lower, "/download"):
			return "/v1/youtube/download", buildChocomilkYoutubeQuery(query)
		default:
			return path, cloneValues(query)
		}
	case ProviderYTDLP:
		switch {
		case strings.HasSuffix(lower, "/search"):
			return "/search/", buildYTDLPYoutubeSearchQuery(query)
		case strings.HasSuffix(lower, "/play"):
			return "/search/", buildYTDLPYoutubeSearchQuery(query)
		case strings.HasSuffix(lower, "/info"):
			return "/info/", buildYTDLPYoutubeInfoQuery(query)
		case strings.HasSuffix(lower, "/download"):
			return "/download/", buildYTDLPYoutubeDownloadQuery(query)
		default:
			return path, cloneValues(query)
		}
	case ProviderNexure:
		switch {
		case strings.HasSuffix(lower, "/search"):
			return "/api/search/youtube", buildNexureYoutubeSearchQuery(query)
		case strings.HasSuffix(lower, "/play"):
			return "/api/search/youtube", buildNexureYoutubeSearchQuery(query)
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func buildYTDLPAIQuery(slug string, query url.Values) url.Values {
	out := url.Values{}
	switch slug {
	case "gemini":
		text := aiPromptValue(query)
		if text != "" {
			out.Set("text", text)
		}
		model := strings.TrimSpace(query.Get("model"))
		if model == "" {
			model = "gemma-3-27b-it"
		}
		out.Set("model", model)
	case "powerbrain":
		if text := aiPromptValue(query); text != "" {
			out.Set("question", text)
		}
	case "felo", "beago":
		if text := aiPromptValue(query); text != "" {
			out.Set("text", text)
		}
	case "deepai-chat":
		if text := aiPromptValue(query); text != "" {
			out.Set("prompt", text)
		}
	default:
		if text := aiPromptValue(query); text != "" {
			out.Set("text", text)
		}
		if model := strings.TrimSpace(query.Get("model")); model != "" {
			out.Set("model", model)
		}
	}
	return out
}

func aiPromptValue(query url.Values) string {
	for _, key := range []string{"ask", "text", "prompt", "q"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func cloneValues(values url.Values) url.Values {
	out := make(url.Values, len(values))
	for key, vals := range values {
		out[key] = append([]string(nil), vals...)
	}
	return out
}

func (s *Service) writeResponse(w http.ResponseWriter, r *http.Request, resp proxyResponse) {
	s.writeCORS(w, r)
	for key, values := range resp.Headers {
		if strings.EqualFold(key, "Content-Length") || strings.EqualFold(key, "Transfer-Encoding") || strings.EqualFold(key, "Connection") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	publicProvider := publicProviderCode(resp.Provider)
	w.Header().Set("X-IRAG-Provider", publicProvider)
	w.Header().Set("X-IRAG-Upstream", publicProvider)
	w.Header().Set("X-IRAG-Fallback-Used", strconvBool(len(resp.FallbackChain) > 1))
	w.Header().Set("X-IRAG-Latency-MS", strconvInt(int(resp.Latency/time.Millisecond)))
	if resp.Raw {
		if resp.ContentType != "" {
			w.Header().Set("Content-Type", resp.ContentType)
		}
		w.WriteHeader(resp.Status)
		_, _ = w.Write(resp.Body)
		return
	}

	meta := map[string]any{
		"provider":       publicProvider,
		"providers_used": publicProviderCodes(resp.FallbackChain),
		"fallback_used":  len(resp.FallbackChain) > 1,
		"latency_ms":     int(resp.Latency / time.Millisecond),
		"cached":         false,
		"cache_ttl":      0,
		"cache_status":   "miss",
	}
	body, err := buildEnvelope(resp.Status, normalizeJSONBody(resp.Body), meta, "")
	if err != nil {
		writeEnvelopeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{"message": err.Error()},
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)
	_, _ = w.Write(body)
}

func (s *Service) writeCached(w http.ResponseWriter, r *http.Request, cached CachedResponse, ttl time.Duration) {
	s.writeCORS(w, r)
	publicProvider := publicProviderCode(cached.Provider)
	w.Header().Set("X-IRAG-Provider", publicProvider)
	w.Header().Set("X-IRAG-Upstream", publicProvider)
	w.Header().Set("X-IRAG-Fallback-Used", "false")
	w.Header().Set("X-IRAG-Cache", "HIT")
	w.Header().Set("X-IRAG-Latency-MS", "0")
	if cached.ContentType != "" {
		w.Header().Set("Content-Type", cached.ContentType)
	}
	w.WriteHeader(cached.Status)
	_, _ = w.Write(cached.Body)
}

func (s *Service) writeFailure(w http.ResponseWriter, status int, spec routeSpec, attempted []string, err error) {
	writeEnvelopeJSON(w, status, map[string]any{
		"ok":   false,
		"code": status,
		"error": map[string]any{
			"message":  err.Error(),
			"upstream": strings.Join(publicProviderCodes(attempted), " -> "),
		},
		"meta": map[string]any{
			"provider":       "",
			"providers_used": publicProviderCodes(attempted),
			"fallback_used":  len(attempted) > 1,
			"latency_ms":     0,
			"cached":         false,
			"cache_ttl":      int(spec.CacheTTL / time.Second),
			"cache_status":   "miss",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Service) handleToBase64(w http.ResponseWriter, r *http.Request) bool {
	lower := strings.ToLower(r.URL.Path)
	if lower != "/v1/tobase64" && lower != "/v1/tools/tobase64" {
		return false
	}
	if r.Method != http.MethodPost {
		writeEnvelopeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": map[string]any{"message": "method not allowed"},
		})
		return true
	}

	filename, contentType, data, err := readToBase64Payload(r)
	if err != nil {
		writeEnvelopeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]any{"message": err.Error()},
		})
		return true
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	result := map[string]any{
		"base64":       encoded,
		"filename":     filename,
		"content_type": contentType,
		"size":         len(data),
	}
	payload, err := buildEnvelope(http.StatusOK, result, map[string]any{
		"provider":   "local",
		"latency_ms": 0,
		"cached":     false,
		"cache_ttl":  0,
	}, "")
	if err != nil {
		writeEnvelopeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{"message": err.Error()},
		})
		return true
	}
	s.writeCORS(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-IRAG-Provider", "local")
	w.Header().Set("X-IRAG-Upstream", "local")
	w.Header().Set("X-IRAG-Fallback-Used", "false")
	w.Header().Set("X-IRAG-Latency-MS", "0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	return true
}

func readToBase64Payload(r *http.Request) (string, string, []byte, error) {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return "", "", nil, err
		}
		if r.MultipartForm != nil {
			for field, headers := range r.MultipartForm.File {
				if len(headers) == 0 {
					continue
				}
				fh := headers[0]
				file, err := fh.Open()
				if err != nil {
					return "", "", nil, err
				}
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					return "", "", nil, err
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" {
					ct = http.DetectContentType(data)
				}
				name := fh.Filename
				if name == "" {
					name = field
				}
				return name, ct, data, nil
			}
		}
		return "", "", nil, errors.New("missing file upload")
	}

	if content := strings.TrimSpace(r.FormValue("content")); content != "" {
		return "content.txt", "text/plain; charset=utf-8", []byte(content), nil
	}
	if body, err := io.ReadAll(r.Body); err == nil && len(body) > 0 {
		ct := strings.TrimSpace(r.Header.Get("Content-Type"))
		if ct == "" {
			ct = http.DetectContentType(body)
		}
		return "body.bin", ct, body, nil
	} else if err != nil {
		return "", "", nil, err
	}
	return "", "", nil, errors.New("missing file or content payload")
}

func (s *Service) log(r *http.Request, spec routeSpec, provider string, attempted []string, status string, httpStatus int, latency time.Duration, size int, cacheKey string, cacheTTL time.Duration, errorCode, errorMessage string, cached bool) {
	if s.logs == nil {
		return
	}
	clientID := strings.TrimSpace(r.Header.Get("X-Client-Id"))
	if clientID == "" {
		clientID = strings.TrimSpace(r.RemoteAddr)
	}
	s.logs.Insert(r.Context(), RequestLog{
		Endpoint:          r.URL.Path,
		Category:          spec.Category,
		ProviderUsed:      provider,
		FallbackChain:     attempted,
		Status:            status,
		HTTPStatus:        httpStatus,
		LatencyMS:         int(latency / time.Millisecond),
		ResponseSizeBytes: size,
		CacheKey:          cacheKey,
		CacheTTLSeconds:   int(cacheTTL / time.Second),
		ErrorCode:         errorCode,
		ErrorMessage:      errorMessage,
		ClientID:          clientID,
		IsPremium:         strings.EqualFold(r.Header.Get("X-Plan"), "premium"),
	})
}

func (s *Service) routeSpecForPath(path string) routeSpec {
	lower := strings.ToLower(path)
	switch {
	case strings.HasPrefix(lower, "/v1/ai/text/"):
		return routeSpec{Category: "ai", Providers: aiTextChain(lower), CacheTTL: 0, Timeout: s.cfg.Timeout}
	case lower == "/v1/ai/generate":
		return routeSpec{Category: "ai", Providers: []ProviderName{ProviderKanata}, CacheTTL: 10 * time.Minute, Timeout: 7 * time.Minute}
	case lower == "/v1/ai/image":
		return routeSpec{Category: "ai", Providers: []ProviderName{ProviderKanata}, CacheTTL: 10 * time.Minute, Timeout: 7 * time.Minute}
	case strings.HasPrefix(lower, "/v1/ai/image/"):
		return routeSpec{Category: "ai", Providers: aiImageChain(lower), CacheTTL: 10 * time.Minute, Timeout: 7 * time.Minute}
	case strings.HasPrefix(lower, "/v1/ai/process/"):
		return routeSpec{Category: "ai", Providers: aiProcessChain(lower), CacheTTL: 10 * time.Minute, Timeout: 7 * time.Minute}
	case strings.HasPrefix(lower, "/v1/i2i/"):
		return routeSpec{Category: "ai", Providers: aiProcessChain(lower), CacheTTL: 10 * time.Minute, Timeout: 7 * time.Minute}
	case strings.HasPrefix(lower, "/v1/ai/"):
		slug := strings.TrimPrefix(lower, "/v1/ai/")
		if slug == "" {
			return routeSpec{}
		}
		if isDirectAIImageSlug(slug) {
			return routeSpec{Category: "ai", Providers: aiDirectImageChain(lower), CacheTTL: 10 * time.Minute, Timeout: 7 * time.Minute}
		}
		if !isAllowedDirectAITextSlug(slug) {
			return routeSpec{}
		}
		return routeSpec{Category: "ai", Providers: aiDirectTextChain(lower), CacheTTL: 0, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/youtube/"):
		slug := strings.TrimPrefix(lower, "/v1/youtube/")
		if !isAllowedYouTubeSlug(slug) {
			return routeSpec{}
		}
		return routeSpec{Category: "youtube", Providers: youtubeChain(lower), CacheTTL: 10 * time.Minute, Timeout: s.cfg.Timeout}
	case lower == "/v1/llm/chatgpt/completions":
		return routeSpec{Category: "llm", Providers: llmChain(lower), CacheTTL: 0, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/download/youtube/search"):
		return routeSpec{Category: "search", Providers: []ProviderName{ProviderRyzumi, ProviderNexure}, CacheTTL: 5 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/download/youtube/playlist"), strings.HasPrefix(lower, "/v1/download/youtube/subtitle"), strings.HasPrefix(lower, "/v1/download/spotify/playlist"), strings.HasPrefix(lower, "/v1/download/tiktok/hd"), strings.HasPrefix(lower, "/v1/download/douyin"), strings.HasPrefix(lower, "/v1/download/applemusic"), strings.HasPrefix(lower, "/v1/download/mediafire"):
		return routeSpec{Category: "download", Providers: downloadChain(lower), CacheTTL: 30 * time.Minute, Timeout: 7 * time.Minute}
	case strings.HasPrefix(lower, "/v1/download/"):
		return routeSpec{Category: "download", Providers: downloadChain(lower), CacheTTL: 30 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/search/"):
		return routeSpec{Category: "search", Providers: searchChain(lower), CacheTTL: 5 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/tools/"):
		switch {
		case strings.Contains(lower, "/screenshot"), strings.Contains(lower, "/removebg"), strings.Contains(lower, "/shorturl"), strings.Contains(lower, "/listbank"), strings.Contains(lower, "/cekbank"), strings.Contains(lower, "/distance"), strings.Contains(lower, "/kurs"), strings.Contains(lower, "/gsmarena"), strings.Contains(lower, "/iphonechat"), strings.Contains(lower, "/brat/animated"), strings.Contains(lower, "/utility/upscale"), strings.Contains(lower, "/tools/upscale"):
			return routeSpec{Category: "tools", Providers: toolsChain(lower), CacheTTL: 1 * time.Minute, Timeout: 7 * time.Minute}
		default:
			return routeSpec{Category: "tools", Providers: toolsChain(lower), CacheTTL: 1 * time.Minute, Timeout: s.cfg.Timeout}
		}
	case lower == "/v1/utility/upscale" || lower == "/v1/tools/upscale":
		return routeSpec{Category: "tools", Providers: []ProviderName{ProviderNexure, ProviderRyzumi}, CacheTTL: 1 * time.Minute, Timeout: 7 * time.Minute}
	case strings.HasPrefix(lower, "/v1/stalk/"):
		return routeSpec{Category: "stalk", Providers: stalkChain(lower), CacheTTL: 5 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/bmkg/"):
		return routeSpec{Category: "bmkg", Providers: []ProviderName{ProviderKanata}, CacheTTL: 60 * time.Second, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/islamic/"):
		return routeSpec{Category: "islamic", Providers: islamicChain(lower), CacheTTL: 1 * time.Hour, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/anime/"):
		return routeSpec{Category: "anime", Providers: []ProviderName{ProviderNexure, ProviderKanata, ProviderRyzumi}, CacheTTL: 15 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/manga/"):
		return routeSpec{Category: "manga", Providers: []ProviderName{ProviderKanata, ProviderNexure}, CacheTTL: 15 * time.Minute, Timeout: s.cfg.Timeout}
	case lower == "/v1/otakudesu" || strings.HasPrefix(lower, "/v1/otakudesu/"):
		return routeSpec{Category: "anime", Providers: []ProviderName{ProviderKanata, ProviderNexure}, CacheTTL: 15 * time.Minute, Timeout: s.cfg.Timeout}
	case lower == "/v1/komiku" || strings.HasPrefix(lower, "/v1/komiku/"):
		return routeSpec{Category: "manga", Providers: []ProviderName{ProviderKanata, ProviderNexure}, CacheTTL: 15 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/weebs/"):
		return routeSpec{Category: "weebs", Providers: []ProviderName{ProviderRyzumi}, CacheTTL: 15 * time.Minute, Timeout: s.cfg.Timeout}
	case lower == "/v1/novel" || strings.HasPrefix(lower, "/v1/novel/"):
		return routeSpec{Category: "novel", Providers: []ProviderName{ProviderChocomilk}, CacheTTL: 15 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/film/"), strings.HasPrefix(lower, "/v1/drama/"), strings.HasPrefix(lower, "/v1/lk21"):
		return routeSpec{Category: "film", Providers: []ProviderName{ProviderKanata, ProviderNexure}, CacheTTL: 10 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/game/"):
		return routeSpec{Category: "game", Providers: gameChain(lower), CacheTTL: 30 * time.Second, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/news/"), lower == "/v1/media" || strings.HasPrefix(lower, "/v1/media/"):
		return routeSpec{Category: "news", Providers: []ProviderName{ProviderKanata, ProviderNexure}, CacheTTL: 10 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/dramabox"):
		return routeSpec{Category: "dramabox", Providers: []ProviderName{ProviderNexure}, CacheTTL: 15 * time.Minute, Timeout: s.cfg.Timeout}
	case strings.HasPrefix(lower, "/v1/tools/cctv"), strings.HasPrefix(lower, "/v1/bsw/cctv"):
		return routeSpec{Category: "bsw", Providers: []ProviderName{ProviderNexure}, CacheTTL: 5 * time.Minute, Timeout: s.cfg.Timeout}
	case lower == "/v1/misc/server-info" || lower == "/v1/server-info":
		return routeSpec{Category: "misc", Providers: []ProviderName{ProviderNexure}, CacheTTL: 5 * time.Minute, Timeout: s.cfg.Timeout}
	case lower == "/v1/misc/ip-whitelist-check":
		return routeSpec{Category: "misc", Providers: []ProviderName{ProviderRyzumi}, CacheTTL: 5 * time.Minute, Timeout: s.cfg.Timeout}
	case lower == "/v1/upload" || strings.HasPrefix(lower, "/v1/upload/"):
		return routeSpec{Category: "upload", Providers: []ProviderName{ProviderNexure, ProviderKanata, ProviderRyzumi}, CacheTTL: 10 * time.Minute, Timeout: s.cfg.Timeout}
	default:
		return routeSpec{}
	}
}

func aiTextChain(path string) []ProviderName {
	switch {
	case strings.Contains(path, "mistral"):
		return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderChocomilk}
	case strings.Contains(path, "gemini") && !strings.Contains(path, "gemini-ryz"):
		return []ProviderName{ProviderYTDLP, ProviderNexure, ProviderRyzumi, ProviderChocomilk}
	case strings.Contains(path, "chocomilk"):
		return []ProviderName{ProviderChocomilk, ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "deepseek-ryz"), strings.Contains(path, "gemini-ryz"), strings.Contains(path, "qwen"), strings.Contains(path, "chatgpt-ryz"):
		return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderChocomilk}
	case strings.Contains(path, "copilot"), strings.Contains(path, "gpt-v2"), strings.Contains(path, "gpt"), strings.Contains(path, "claila"), strings.Contains(path, "meta"), strings.Contains(path, "perplexity"), strings.Contains(path, "z-ai"), strings.Contains(path, "webpilot"), strings.Contains(path, "ai4chat"):
		return []ProviderName{ProviderNexure, ProviderRyzumi, ProviderChocomilk}
	default:
		return []ProviderName{ProviderNexure, ProviderRyzumi, ProviderChocomilk}
	}
}

func aiImageChain(path string) []ProviderName {
	if strings.Contains(path, "kanata") {
		return []ProviderName{ProviderKanata, ProviderNexure, ProviderRyzumi}
	}
	return []ProviderName{ProviderNexure, ProviderKanata, ProviderRyzumi}
}

func aiProcessChain(path string) []ProviderName {
	switch {
	case strings.Contains(path, "tololi"), strings.Contains(path, "enhance2x"), strings.Contains(path, "nanobanana"), strings.Contains(path, "nano-banana"):
		return []ProviderName{ProviderChocomilk, ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "colorize"), strings.Contains(path, "faceswap"), strings.Contains(path, "upscale"), strings.Contains(path, "enhance"), strings.Contains(path, "removebg"), strings.Contains(path, "waifu2x"), strings.Contains(path, "image2txt"):
		return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderChocomilk}
	case strings.Contains(path, "nsfw-check"):
		return []ProviderName{ProviderNexure, ProviderRyzumi, ProviderChocomilk}
	case strings.Contains(path, "toanime"):
		return []ProviderName{ProviderNexure, ProviderRyzumi, ProviderChocomilk}
	default:
		return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderChocomilk}
	}
}

func downloadChain(path string) []ProviderName {
	switch {
	case strings.Contains(path, "/youtube/"):
		if strings.Contains(path, "/search") {
			return []ProviderName{ProviderRyzumi, ProviderNexure}
		}
		if strings.Contains(path, "/playlist") || strings.Contains(path, "/subtitle") {
			return []ProviderName{ProviderYTDLP, ProviderNexure, ProviderRyzumi}
		}
		return []ProviderName{ProviderKanata, ProviderNexure, ProviderYTDLP, ProviderRyzumi}
	case strings.Contains(path, "/tiktok"):
		if strings.Contains(path, "/hd") || strings.Contains(path, "/douyin") {
			return []ProviderName{ProviderYTDLP, ProviderRyzumi, ProviderNexure}
		}
		return []ProviderName{ProviderNexure, ProviderKanata, ProviderRyzumi, ProviderYTDLP}
	case strings.Contains(path, "/instagram"):
		return []ProviderName{ProviderNexure, ProviderRyzumi, ProviderChocomilk}
	case strings.Contains(path, "/spotify"):
		if strings.Contains(path, "/playlist") {
			return []ProviderName{ProviderYTDLP, ProviderRyzumi}
		}
		return []ProviderName{ProviderNexure, ProviderRyzumi, ProviderYTDLP}
	case strings.Contains(path, "/twitter"):
		return []ProviderName{ProviderChocomilk, ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/threads"):
		return []ProviderName{ProviderChocomilk, ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/soundcloud"):
		return []ProviderName{ProviderNexure, ProviderRyzumi, ProviderYTDLP}
	case strings.Contains(path, "/gdrive"):
		return []ProviderName{ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "/mediafire"), strings.Contains(path, "/applemusic"):
		return []ProviderName{ProviderYTDLP, ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/videy"), strings.Contains(path, "/sfile"), strings.Contains(path, "/shopee/video"), strings.Contains(path, "/nhentai"):
		return []ProviderName{ProviderYTDLP, ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "/douyin"):
		return []ProviderName{ProviderYTDLP, ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/bilibili"), strings.Contains(path, "/bstation"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/tidal"), strings.Contains(path, "/deezer"), strings.Contains(path, "/capcut"):
		return []ProviderName{ProviderChocomilk}
	case strings.Contains(path, "/scribd"):
		return []ProviderName{ProviderNexure}
	case strings.Contains(path, "/mega"), strings.Contains(path, "/terabox"), strings.Contains(path, "/pixeldrain"), strings.Contains(path, "/krakenfiles"), strings.Contains(path, "/danbooru"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/reddit"):
		return []ProviderName{ProviderNexure}
	case strings.Contains(path, "/applemusic"):
		return []ProviderName{ProviderYTDLP}
	default:
		return []ProviderName{ProviderNexure, ProviderKanata, ProviderRyzumi}
	}
}

func islamicChain(path string) []ProviderName {
	switch {
	case strings.Contains(path, "/quran"), strings.Contains(path, "/tafsir"), strings.Contains(path, "/topegon"), strings.Contains(path, "/hadith"):
		return []ProviderName{ProviderYTDLP, ProviderKanata}
	default:
		return []ProviderName{ProviderKanata, ProviderYTDLP}
	}
}

func ytdlpIslamicPathAndQuery(path string, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch {
	case path == "/v1/islamic/quran":
		return "/quran/surah", out
	case strings.HasPrefix(path, "/v1/islamic/quran/"):
		return "/quran/surah/" + strings.TrimPrefix(path, "/v1/islamic/quran/"), out
	case path == "/v1/islamic/tafsir":
		surah := strings.TrimSpace(firstNonEmpty(query.Get("surah"), query.Get("nomor")))
		if surah != "" {
			out = url.Values{}
			out.Set("surah", surah)
		}
		return "/tafsir", out
	case path == "/v1/islamic/topegon":
		text := strings.TrimSpace(firstNonEmpty(query.Get("text"), query.Get("ask"), query.Get("q")))
		if text != "" {
			out = url.Values{}
			out.Set("text", text)
		}
		return "/topegon", out
	case strings.HasPrefix(path, "/v1/islamic/hadith/"):
		trimmed := strings.TrimPrefix(path, "/v1/islamic/hadith/")
		parts := strings.Split(trimmed, "/")
		if len(parts) >= 2 {
			book := strings.TrimSpace(parts[0])
			n := strings.TrimSpace(parts[1])
			out = url.Values{}
			if book != "" {
				out.Set("book", book)
			}
			if n != "" {
				out.Set("hadith_id", n)
			}
		}
		return "/hadits", out
	default:
		return path, out
	}
}

func (s *Service) toolsUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	lower := strings.ToLower(path)
	switch provider {
	case ProviderKanata:
		switch {
		case strings.Contains(lower, "/translate"):
			out := url.Values{}
			if text := strings.TrimSpace(firstNonEmpty(query.Get("text"), query.Get("q"), query.Get("ask"))); text != "" {
				out.Set("text", text)
			}
			if to := strings.TrimSpace(firstNonEmpty(query.Get("to"), query.Get("target"))); to != "" {
				out.Set("to", to)
			}
			if from := strings.TrimSpace(firstNonEmpty(query.Get("from"), query.Get("source"))); from != "" {
				out.Set("from", from)
			}
			return "/googletranslate", out
		case strings.Contains(lower, "/kbbi"):
			out := url.Values{}
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("text"), query.Get("query"))); q != "" {
				out.Set("q", q)
			}
			return "/kbbi", out
		case strings.Contains(lower, "/ipinfo"):
			ip := toolPathTail(lower, "/v1/tools/ipinfo")
			if ip == "" {
				ip = strings.TrimSpace(firstNonEmpty(query.Get("ip"), query.Get("q")))
			}
			if ip != "" {
				return "/ipinfo/" + ip, url.Values{}
			}
			return "/ipinfo", cloneValues(query)
		case strings.Contains(lower, "/carbon"):
			out := url.Values{}
			if code := strings.TrimSpace(firstNonEmpty(query.Get("code"), query.Get("text"))); code != "" {
				out.Set("code", code)
			}
			return "/carbon", out
		default:
			return path, cloneValues(query)
		}
	case ProviderNexure:
		switch {
		case strings.Contains(lower, "/cctv"):
			tail := ""
			switch {
			case strings.HasPrefix(lower, "/v1/tools/cctv"):
				tail = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(lower, "/v1/tools/cctv"), "/"))
			case strings.HasPrefix(lower, "/v1/bsw/cctv"):
				tail = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(lower, "/v1/bsw/cctv"), "/"))
			}
			if tail == "" || tail == "/all" {
				return "/api/bsw/cctv/all", url.Values{}
			}
			if tail == "/search" {
				out := url.Values{}
				if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("text"), query.Get("name"), query.Get("location"))); q != "" {
					out.Set("query", q)
				}
				return "/api/bsw/cctv/search", out
			}
			if strings.HasPrefix(tail, "detail/") {
				id := strings.TrimSpace(strings.TrimPrefix(tail, "detail/"))
				if id != "" {
					return "/api/bsw/cctv/detail/" + id, url.Values{}
				}
			}
			if id := strings.Trim(strings.TrimPrefix(tail, "/"), " "); id != "" {
				return "/api/bsw/cctv/detail/" + id, url.Values{}
			}
			return "/api/bsw/cctv/all", cloneValues(query)
		case strings.Contains(lower, "/dramabox"):
			if strings.HasSuffix(lower, "/search") {
				out := url.Values{}
				if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
					out.Set("q", q)
				}
				return "/api/dramabox/search", out
			}
			return "/api/dramabox", cloneValues(query)
		case strings.Contains(lower, "/server-info"):
			return "/api/misc/server-info", cloneValues(query)
		case strings.Contains(lower, "/weather"):
			out := url.Values{}
			if city := strings.TrimSpace(firstNonEmpty(query.Get("city"), query.Get("q"))); city != "" {
				out.Set("city", city)
			}
			return "/api/info/weather", out
		case strings.Contains(lower, "/cekresi"):
			out := url.Values{}
			if noresi := strings.TrimSpace(firstNonEmpty(query.Get("noresi"), query.Get("resi"), query.Get("tracking_number"))); noresi != "" {
				out.Set("noresi", noresi)
				out.Set("resi", noresi)
			}
			if ekspedisi := strings.TrimSpace(query.Get("ekspedisi")); ekspedisi != "" {
				out.Set("ekspedisi", ekspedisi)
			}
			return "/api/tools/cekresi", out
		case strings.Contains(lower, "/qris-converter"):
			out := url.Values{}
			if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"))); u != "" {
				out.Set("url", u)
			}
			if nominal := strings.TrimSpace(firstNonEmpty(query.Get("nominal"), query.Get("amount"))); nominal != "" {
				out.Set("nominal", nominal)
			}
			return "/api/tool/qris-converter", out
		case strings.Contains(lower, "/qr"):
			out := url.Values{}
			if text := strings.TrimSpace(firstNonEmpty(query.Get("text"), query.Get("q"))); text != "" {
				out.Set("text", text)
			}
			if frame := strings.TrimSpace(query.Get("frame")); frame != "" {
				out.Set("frame", frame)
			}
			return "/api/image/qr", out
		case strings.Contains(lower, "/nsfw"):
			out := url.Values{}
			if target := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"))); target != "" {
				out.Set("url", target)
				out.Set("image_url", target)
			}
			return "/api/tools/nsfw-check", out
		case strings.Contains(lower, "/ssweb"):
			out := url.Values{}
			if target := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("target"))); target != "" {
				out.Set("url", target)
			}
			if mode := strings.TrimSpace(query.Get("mode")); mode != "" {
				out.Set("mode", mode)
			}
			return "/api/tools/ssweb", out
		case strings.Contains(lower, "/utility/upscale"), strings.Contains(lower, "/tools/upscale"):
			out := url.Values{}
			if img := strings.TrimSpace(firstNonEmpty(query.Get("imgUrl"), query.Get("image_url"), query.Get("url"))); img != "" {
				out.Set("imgUrl", img)
				out.Set("image_url", img)
				out.Set("url", img)
			}
			return "/api/tools/upscale", out
		case strings.Contains(lower, "/pln"):
			out := url.Values{}
			if id := strings.TrimSpace(firstNonEmpty(query.Get("id"), query.Get("id_pel"), query.Get("customer_id"))); id != "" {
				out.Set("id", id)
				out.Set("id_pel", id)
			}
			return "/api/tools/pln", out
		case strings.Contains(lower, "/pajak"):
			out := url.Values{}
			if plat := normalizePlate(firstNonEmpty(query.Get("plat"), query.Get("no_pol"), query.Get("plate"))); plat != "" {
				out.Set("plat", plat)
			}
			return "/api/tools/cek-pajak/jabar", out
		case strings.Contains(lower, "/brat/animated"):
			out := url.Values{}
			if text := strings.TrimSpace(firstNonEmpty(query.Get("text"), query.Get("q"))); text != "" {
				out.Set("text", text)
			}
			return "/api/image/brat/animated", out
		case strings.Contains(lower, "/brat"):
			out := url.Values{}
			if text := strings.TrimSpace(firstNonEmpty(query.Get("text"), query.Get("q"))); text != "" {
				out.Set("text", text)
			}
			return "/api/image/brat", out
		case strings.Contains(lower, "/whois"):
			out := url.Values{}
			if domain := strings.TrimSpace(firstNonEmpty(query.Get("domain"), query.Get("q"))); domain != "" {
				out.Set("domain", domain)
			}
			return "/api/tool/whois", out
		case strings.Contains(lower, "/check-hosting"):
			out := url.Values{}
			if domain := strings.TrimSpace(firstNonEmpty(query.Get("domain"), query.Get("q"))); domain != "" {
				out.Set("domain", domain)
			}
			return "/api/tool/check-hosting", out
		case strings.Contains(lower, "/hargapangan"):
			return "/api/tool/hargapangan", cloneValues(query)
		case strings.Contains(lower, "/mc-lookup"):
			out := url.Values{}
			if ip := strings.TrimSpace(firstNonEmpty(query.Get("ip"), query.Get("q"))); ip != "" {
				out.Set("ip", ip)
			}
			return "/api/tool/mc-lookup", out
		case strings.Contains(lower, "/qris-converter"):
			out := url.Values{}
			if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"))); u != "" {
				out.Set("url", u)
			}
			if nominal := strings.TrimSpace(firstNonEmpty(query.Get("nominal"), query.Get("amount"))); nominal != "" {
				out.Set("nominal", nominal)
			}
			return "/api/tool/qris-converter", out
		case strings.Contains(lower, "/turnstile-bypass"):
			out := url.Values{}
			if sitekey := strings.TrimSpace(firstNonEmpty(query.Get("sitekey"), query.Get("url"))); sitekey != "" {
				out.Set("sitekey", sitekey)
				out.Set("url", sitekey)
			}
			return "/api/tools/turnstile-min", out
		case strings.Contains(lower, "/turnstile/sitekey"):
			out := url.Values{}
			if sitekey := strings.TrimSpace(firstNonEmpty(query.Get("sitekey"), query.Get("url"))); sitekey != "" {
				out.Set("sitekey", sitekey)
				out.Set("url", sitekey)
			}
			return "/api/tool/turnstile/sitekey", out
		default:
			return path, cloneValues(query)
		}
	case ProviderYTDLP:
		switch {
		case strings.Contains(lower, "/screenshot"):
			out := url.Values{}
			if target := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("target"))); target != "" {
				out.Set("url", target)
			}
			if mode := strings.TrimSpace(query.Get("mode")); mode != "" {
				out.Set("mode", mode)
			}
			return "/screenshot", out
		case strings.Contains(lower, "/removebg"):
			out := cloneValues(query)
			if imageURL := strings.TrimSpace(firstNonEmpty(query.Get("image_url"), query.Get("url"))); imageURL != "" {
				out.Set("image_url", imageURL)
				out.Set("mode", "url")
			}
			return "/removebg", out
		case strings.Contains(lower, "/shorturl"):
			out := url.Values{}
			if target := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("long_url"), query.Get("target"))); target != "" {
				out.Set("url", target)
				out.Set("long_url", target)
			}
			return "/shorturl", out
		case strings.Contains(lower, "/subdofinder"):
			out := url.Values{}
			if domain := strings.TrimSpace(firstNonEmpty(query.Get("domain"), query.Get("q"))); domain != "" {
				out.Set("domain", domain)
			}
			return "/subdofinder", out
		case strings.Contains(lower, "/yt-transcript"):
			out := url.Values{}
			if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("video_url"), query.Get("link"))); u != "" {
				out.Set("url", u)
			}
			return "/api/tool/yt-transcript", out
		case strings.Contains(lower, "/isrc"):
			out := url.Values{}
			if isrc := strings.TrimSpace(firstNonEmpty(query.Get("isrc"), query.Get("q"), query.Get("query"))); isrc != "" {
				out.Set("isrc", isrc)
				out.Set("q", isrc)
			}
			return "/api/tool/isrc", out
		case strings.Contains(lower, "/listbank"):
			return "/listbank", cloneValues(query)
		case strings.Contains(lower, "/cekbank"):
			out := url.Values{}
			if bank := strings.TrimSpace(firstNonEmpty(query.Get("bank_code"), query.Get("bank"), query.Get("kode_bank"))); bank != "" {
				out.Set("bank_code", bank)
				out.Set("bank", bank)
			}
			if account := strings.TrimSpace(firstNonEmpty(query.Get("account_number"), query.Get("no_rek"), query.Get("rekening"))); account != "" {
				out.Set("account_number", account)
				out.Set("no_rek", account)
			}
			return "/cekbank", out
		case strings.Contains(lower, "/distance"):
			out := url.Values{}
			if from := strings.TrimSpace(firstNonEmpty(query.Get("from"), query.Get("dari"), query.Get("origin"))); from != "" {
				out.Set("dari", from)
				out.Set("from", from)
			}
			if to := strings.TrimSpace(firstNonEmpty(query.Get("to"), query.Get("ke"), query.Get("destination"))); to != "" {
				out.Set("ke", to)
				out.Set("to", to)
			}
			return "/jarak", out
		case strings.Contains(lower, "/kurs"):
			out := url.Values{}
			if from := strings.TrimSpace(firstNonEmpty(query.Get("dari"), query.Get("from"))); from != "" {
				out.Set("dari", from)
			}
			if to := strings.TrimSpace(firstNonEmpty(query.Get("ke"), query.Get("to"))); to != "" {
				out.Set("ke", to)
			}
			if amount := strings.TrimSpace(firstNonEmpty(query.Get("jumlah"), query.Get("amount"))); amount != "" {
				out.Set("jumlah", amount)
			}
			return "/kurs", out
		case strings.Contains(lower, "/currency-converter"):
			out := url.Values{}
			if from := strings.TrimSpace(firstNonEmpty(query.Get("dari"), query.Get("from"))); from != "" {
				out.Set("dari", from)
			}
			if to := strings.TrimSpace(firstNonEmpty(query.Get("ke"), query.Get("to"))); to != "" {
				out.Set("ke", to)
			}
			if amount := strings.TrimSpace(firstNonEmpty(query.Get("jumlah"), query.Get("amount"))); amount != "" {
				out.Set("jumlah", amount)
			}
			return "/kurs", out
		case strings.Contains(lower, "/gsmarena"):
			out := url.Values{}
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("device"))); q != "" {
				out.Set("q", q)
				out.Set("device", q)
			}
			return "/gsmarena", out
		case strings.Contains(lower, "/iphonechat"):
			out := url.Values{}
			if text := strings.TrimSpace(firstNonEmpty(query.Get("text"), query.Get("q"))); text != "" {
				out.Set("text", text)
			}
			if user := strings.TrimSpace(query.Get("user")); user != "" {
				out.Set("user", user)
			}
			if jam := strings.TrimSpace(query.Get("jam")); jam != "" {
				out.Set("jam", jam)
			}
			if profileURL := strings.TrimSpace(firstNonEmpty(query.Get("profile_url"), query.Get("profileUrl"))); profileURL != "" {
				out.Set("profile_url", profileURL)
			}
			return "/maker/iqc", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func bmkgUpstreamPathAndQuery(path string, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch {
	case path == "/v1/bmkg/earthquake":
		return "/bmkg/gempa", out
	case path == "/v1/bmkg/earthquake/felt":
		return "/bmkg/gempa/dirasakan", out
	case path == "/v1/bmkg/weather":
		provinsi := strings.TrimSpace(firstNonEmpty(query.Get("provinsi"), query.Get("province"), query.Get("slug"), query.Get("q"), query.Get("query")))
		if provinsi != "" {
			out = url.Values{}
			out.Set("provinsi", provinsi)
		}
		return "/bmkg/cuaca", out
	case path == "/v1/bmkg/weather/village":
		adm4 := strings.TrimSpace(firstNonEmpty(query.Get("adm4"), query.Get("code"), query.Get("id"), query.Get("wilayah")))
		if adm4 != "" {
			out = url.Values{}
			out.Set("adm4", adm4)
		}
		return "/bmkg/cuaca/desa", out
	case path == "/v1/bmkg/provinces":
		return "/bmkg/cuaca/provinces", out
	case path == "/v1/bmkg/region/search":
		q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("text"), query.Get("name"), query.Get("wilayah")))
		if q != "" {
			out = url.Values{}
			out.Set("q", q)
		}
		return "/bmkg/wilayah/search", out
	default:
		return path, cloneValues(query)
	}
}

func animeUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch provider {
	case ProviderKanata:
		switch {
		case path == "/v1/anime/home":
			return "/api/otakudesu", out
		case path == "/v1/anime/schedule":
			return "/api/otakudesu/schedule", out
		case path == "/v1/anime/genres":
			return "/api/otakudesu/genre", out
		case strings.HasPrefix(path, "/v1/anime/genre/"):
			genre := strings.TrimSpace(strings.TrimPrefix(path, "/v1/anime/genre/"))
			if genre != "" {
				out = url.Values{}
				out.Set("genre", genre)
			}
			return "/api/otakudesu/animebygenre", out
		case path == "/v1/anime/search":
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("q", q)
			}
			return "/api/otakudesu/search", out
		case strings.HasPrefix(path, "/v1/anime/batch/"):
			slug := strings.TrimSpace(strings.TrimPrefix(path, "/v1/anime/batch/"))
			if slug != "" {
				out = url.Values{}
				out.Set("slug", slug)
			}
			return "/api/otakudesu/download/batch", out
		case path == "/v1/anime/iframe":
			if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("embed_url"), query.Get("iframe"))); u != "" {
				out = url.Values{}
				out.Set("url", u)
			}
			return "/api/otakudesu/getiframe", out
		case path == "/v1/anime/nonce":
			return "/api/otakudesu/nonce", out
		case strings.HasPrefix(path, "/v1/anime/detail/"):
			slug := strings.TrimPrefix(path, "/v1/anime/detail/")
			return "/api/otakudesu/detail/" + slug, out
		case strings.HasPrefix(path, "/v1/anime/episode/"):
			slug := strings.TrimPrefix(path, "/v1/anime/episode/")
			return "/api/otakudesu/episode/" + slug, out
		case strings.HasPrefix(path, "/v1/anime/full/"):
			slug := strings.TrimPrefix(path, "/v1/anime/full/")
			return "/api/otakudesu/lengkap/" + slug, out
		default:
			return path, cloneValues(query)
		}
	case ProviderNexure:
		switch {
		case path == "/v1/anime/home":
			return "/api/otakudesu", out
		case path == "/v1/anime/schedule":
			return "/api/otakudesu/schedule", out
		case path == "/v1/anime/genres":
			return "/api/otakudesu/genre", out
		case strings.HasPrefix(path, "/v1/anime/genre/"):
			genre := strings.TrimSpace(strings.TrimPrefix(path, "/v1/anime/genre/"))
			if genre != "" {
				out = url.Values{}
				out.Set("genre", genre)
			}
			return "/api/otakudesu/animebygenre", out
		case path == "/v1/anime/search":
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("q", q)
			}
			return "/api/otakudesu/search", out
		case strings.HasPrefix(path, "/v1/anime/detail/"):
			slug := strings.TrimPrefix(path, "/v1/anime/detail/")
			return "/api/otakudesu/detail/" + slug, out
		case strings.HasPrefix(path, "/v1/anime/episode/"):
			slug := strings.TrimPrefix(path, "/v1/anime/episode/")
			return "/api/otakudesu/episode/" + slug, out
		case strings.HasPrefix(path, "/v1/anime/full/"):
			slug := strings.TrimPrefix(path, "/v1/anime/full/")
			return "/api/otakudesu/lengkap/" + slug, out
		case path == "/v1/anime/nonce":
			return "/api/otakudesu/nonce", out
		case path == "/v1/anime/iframe":
			if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("embed_url"), query.Get("iframe"))); u != "" {
				out = url.Values{}
				out.Set("url", u)
			}
			return "/api/otakudesu/getiframe", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func mangaUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch provider {
	case ProviderKanata:
		switch {
		case path == "/v1/manga/search":
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("q", q)
			}
			return "/api/komiku/search", out
		case strings.HasPrefix(path, "/v1/manga/chapter/"):
			slug := strings.TrimSpace(strings.TrimPrefix(path, "/v1/manga/chapter/"))
			if slug != "" {
				out = url.Values{}
				out.Set("slug", slug)
			}
			return "/api/komiku/chapter", out
		case strings.HasPrefix(path, "/v1/manga/detail/"):
			slug := strings.TrimPrefix(path, "/v1/manga/detail/")
			return "/api/komiku/detail/" + slug, out
		case path == "/v1/manga/latest":
			return "/api/komiku/terbaru", out
		default:
			return path, cloneValues(query)
		}
	case ProviderNexure:
		switch {
		case path == "/v1/manga/search":
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("q", q)
			}
			return "/api/komiku/search", out
		case strings.HasPrefix(path, "/v1/manga/chapter/"):
			slug := strings.TrimSpace(strings.TrimPrefix(path, "/v1/manga/chapter/"))
			if slug != "" {
				out = url.Values{}
				out.Set("slug", slug)
			}
			return "/api/komiku/chapter", out
		case strings.HasPrefix(path, "/v1/manga/detail/"):
			slug := strings.TrimPrefix(path, "/v1/manga/detail/")
			return "/api/komiku/detail/" + slug, out
		case path == "/v1/manga/latest":
			return "/api/komiku/latest", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func otakudesuAliasPath(path string) string {
	switch {
	case path == "/v1/otakudesu", path == "/v1/otakudesu/home":
		return "/v1/anime/home"
	case path == "/v1/otakudesu/schedule":
		return "/v1/anime/schedule"
	case path == "/v1/otakudesu/genres":
		return "/v1/anime/genres"
	case path == "/v1/otakudesu/genre":
		return "/v1/anime/genres"
	case strings.HasPrefix(path, "/v1/otakudesu/genre/"):
		return "/v1/anime/genre/" + strings.TrimPrefix(path, "/v1/otakudesu/genre/")
	case path == "/v1/otakudesu/search":
		return "/v1/anime/search"
	case strings.HasPrefix(path, "/v1/otakudesu/batch/"):
		return "/v1/anime/batch/" + strings.TrimPrefix(path, "/v1/otakudesu/batch/")
	case path == "/v1/otakudesu/iframe":
		return "/v1/anime/iframe"
	case path == "/v1/otakudesu/nonce":
		return "/v1/anime/nonce"
	case strings.HasPrefix(path, "/v1/otakudesu/detail/"):
		return "/v1/anime/detail/" + strings.TrimPrefix(path, "/v1/otakudesu/detail/")
	case strings.HasPrefix(path, "/v1/otakudesu/episode/"):
		return "/v1/anime/episode/" + strings.TrimPrefix(path, "/v1/otakudesu/episode/")
	case strings.HasPrefix(path, "/v1/otakudesu/lengkap/"):
		return "/v1/anime/full/" + strings.TrimPrefix(path, "/v1/otakudesu/lengkap/")
	case strings.HasPrefix(path, "/v1/otakudesu/animebygenre/"):
		return "/v1/anime/genre/" + strings.TrimPrefix(path, "/v1/otakudesu/animebygenre/")
	default:
		return "/v1/anime" + strings.TrimPrefix(path, "/v1/otakudesu")
	}
}

func komikuAliasPath(path string) string {
	switch {
	case path == "/v1/komiku", path == "/v1/komiku/home", path == "/v1/komiku/latest":
		return "/v1/manga/latest"
	case path == "/v1/komiku/search":
		return "/v1/manga/search"
	case strings.HasPrefix(path, "/v1/komiku/chapter/"):
		return "/v1/manga/chapter/" + strings.TrimPrefix(path, "/v1/komiku/chapter/")
	case strings.HasPrefix(path, "/v1/komiku/detail/"):
		return "/v1/manga/detail/" + strings.TrimPrefix(path, "/v1/komiku/detail/")
	default:
		return "/v1/manga" + strings.TrimPrefix(path, "/v1/komiku")
	}
}

func filmUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch provider {
	case ProviderKanata:
		switch {
		case path == "/v1/film/search", path == "/v1/drama/search":
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("q", q)
			}
			return "/api/nontonfilm/search", out
		case strings.HasPrefix(path, "/v1/film/detail/"), strings.HasPrefix(path, "/v1/drama/detail/"):
			slug := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(path, "/v1/film/detail/"), "/v1/drama/detail/"))
			return "/api/nontonfilm/detail/" + slug, out
		case path == "/v1/film/stream":
			if id := strings.TrimSpace(firstNonEmpty(query.Get("id"), query.Get("slug"), query.Get("url"))); id != "" {
				out = url.Values{}
				out.Set("id", id)
			}
			return "/api/nontonfilm/stream", out
		case path == "/v1/film/home", path == "/v1/film", path == "/v1/drama/home", path == "/v1/drama":
			return "/api/nontonfilm", out
		default:
			return path, cloneValues(query)
		}
	case ProviderNexure:
		switch {
		case path == "/v1/lk21", path == "/v1/film/home", path == "/v1/drama/home":
			return "/api/lk21", out
		case strings.HasPrefix(path, "/v1/lk21/episode/"):
			slug := strings.TrimPrefix(path, "/v1/lk21/episode/")
			return "/api/lk21/episode/" + slug, out
		case strings.HasPrefix(path, "/v1/film/detail/"), strings.HasPrefix(path, "/v1/drama/detail/"):
			slug := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(path, "/v1/film/detail/"), "/v1/drama/detail/"))
			return "/api/lk21/detail/" + slug, out
		case path == "/v1/film/search", path == "/v1/drama/search":
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
				out = url.Values{}
				out.Set("q", q)
			}
			return "/api/lk21/search", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func newsUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch provider {
	case ProviderKanata:
		switch {
		case path == "/v1/news/top":
			return "/news/top", out
		default:
			return path, cloneValues(query)
		}
	case ProviderNexure:
		switch {
		case path == "/v1/news/cnn":
			return "/api/info/cnn", out
		case path == "/v1/news/top":
			return "/api/info/news", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func novelUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch provider {
	case ProviderChocomilk:
		switch {
		case path == "/v1/novel", path == "/v1/novel/home":
			return "/v1/novel/home", out
		case path == "/v1/novel/hot", path == "/v1/novel/hot-search":
			return "/v1/novel/hot-search", out
		case path == "/v1/novel/search":
			if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("ask"), query.Get("text"))); q != "" {
				out = url.Values{}
				out.Set("q", q)
				out.Set("query", q)
			}
			if page := strings.TrimSpace(query.Get("page")); page != "" {
				out.Set("page", page)
			}
			return "/v1/novel/search", out
		case path == "/v1/novel/genre":
			if genre := strings.TrimSpace(firstNonEmpty(query.Get("genre"), query.Get("q"), query.Get("query"), query.Get("title"))); genre != "" {
				out = url.Values{}
				out.Set("genre", genre)
				out.Set("q", genre)
			}
			if page := strings.TrimSpace(query.Get("page")); page != "" {
				out.Set("page", page)
			}
			return "/v1/novel/genre", out
		case path == "/v1/novel/chapters":
			if chapterURL := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("chapter_url"), query.Get("href"), query.Get("link"))); chapterURL != "" {
				out = url.Values{}
				out.Set("url", chapterURL)
			}
			return "/v1/novel/chapters", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func cctvUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	if provider != ProviderNexure {
		return path, out
	}

	lower := strings.ToLower(path)
	tail := ""
	switch {
	case strings.HasPrefix(lower, "/v1/tools/cctv"):
		tail = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(lower, "/v1/tools/cctv"), "/"))
	case strings.HasPrefix(lower, "/v1/bsw/cctv"):
		tail = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(lower, "/v1/bsw/cctv"), "/"))
	}

	switch {
	case tail == "" || tail == "all":
		return "/api/bsw/cctv/all", out
	case tail == "search":
		if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("text"), query.Get("name"), query.Get("location"))); q != "" {
			out = url.Values{}
			out.Set("query", q)
		}
		return "/api/bsw/cctv/search", out
	case strings.HasPrefix(tail, "detail/"):
		id := strings.TrimSpace(strings.TrimPrefix(tail, "detail/"))
		if id != "" {
			return "/api/bsw/cctv/detail/" + id, out
		}
	case tail != "":
		return "/api/bsw/cctv/detail/" + strings.TrimPrefix(tail, "/"), out
	}
	return "/api/bsw/cctv/all", out
}

func dramaboxUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	if provider != ProviderNexure {
		return path, out
	}
	if strings.HasSuffix(strings.ToLower(path), "/search") {
		if q := strings.TrimSpace(firstNonEmpty(query.Get("q"), query.Get("query"), query.Get("title"))); q != "" {
			out = url.Values{}
			out.Set("q", q)
		}
		return "/api/dramabox/search", out
	}
	return "/api/dramabox", out
}

func serverInfoUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	if provider != ProviderNexure {
		return path, out
	}
	return "/api/misc/server-info", out
}

func miscUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch path {
	case "/v1/misc/server-info":
		if provider != ProviderNexure {
			return path, out
		}
		return "/api/misc/server-info", out
	case "/v1/misc/ip-whitelist-check":
		if provider != ProviderRyzumi {
			return path, out
		}
		if ip := strings.TrimSpace(firstNonEmpty(query.Get("ip"), query.Get("q"), query.Get("query"))); ip != "" {
			out = url.Values{}
			out.Set("ip", ip)
		}
		return "/api/misc/ip-whitelist-check", out
	default:
		return path, out
	}
}

func gameChain(path string) []ProviderName {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "/growagarden/stock") {
		return []ProviderName{ProviderNexure, ProviderRyzumi}
	}
	if strings.Contains(lower, "/growagarden/") {
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	}
	return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderKanata}
}

func gameUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	lower := strings.ToLower(path)
	if !strings.HasPrefix(lower, "/v1/game/growagarden/") {
		return path, cloneValues(query)
	}

	sub := strings.TrimPrefix(lower, "/v1/game/growagarden/")
	switch sub {
	case "stock":
		switch provider {
		case ProviderNexure:
			return "/api/info/growagarden", out
		case ProviderRyzumi:
			return "/api/tool/growagarden", out
		default:
			return path, cloneValues(query)
		}
	case "crops", "pets", "gear", "eggs", "cosmetics", "events":
		if provider == ProviderRyzumi {
			out = url.Values{}
			out.Set("category", sub)
			out.Set("type", sub)
			return "/api/tool/growagarden", out
		}
		if provider == ProviderNexure {
			out = url.Values{}
			out.Set("category", sub)
			return "/api/info/growagarden", out
		}
		return path, cloneValues(query)
	default:
		return path, cloneValues(query)
	}
}

func mediaUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	switch provider {
	case ProviderKanata:
		switch {
		case path == "/v1/media/tv":
			return "/tv/now", out
		case path == "/v1/media", path == "/v1/media/":
			return "/tv/now", out
		default:
			return path, cloneValues(query)
		}
	case ProviderNexure:
		switch {
		case path == "/v1/media/tv":
			return "/api/info/tv", out
		case path == "/v1/media", path == "/v1/media/":
			return "/api/info/tv", out
		default:
			return path, cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func weebsUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	out := cloneValues(query)
	if provider != ProviderRyzumi {
		return path, out
	}

	lower := strings.ToLower(path)
	switch {
	case lower == "/v1/weebs/anime-info":
		q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title")))
		if q != "" {
			out = url.Values{}
			out.Set("query", q)
		}
		return "/api/weebs/anime-info", out
	case lower == "/v1/weebs/manga-info":
		q := strings.TrimSpace(firstNonEmpty(query.Get("query"), query.Get("q"), query.Get("title")))
		if q != "" {
			out = url.Values{}
			out.Set("query", q)
		}
		return "/api/weebs/manga-info", out
	case lower == "/v1/weebs/sfw-waifu":
		tag := strings.TrimSpace(firstNonEmpty(query.Get("tag"), query.Get("q"), query.Get("query")))
		if tag != "" {
			out = url.Values{}
			out.Set("tag", tag)
		}
		return "/api/weebs/sfw-waifu", out
	case lower == "/v1/weebs/whatanime":
		if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"), query.Get("imgUrl"))); u != "" {
			out = url.Values{}
			out.Set("url", u)
		}
		return "/api/weebs/whatanime", out
	default:
		return path, out
	}
}

func uploadUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	switch provider {
	case ProviderNexure:
		switch {
		case path == "/v1/upload":
			return "/api/upload", cloneValues(query)
		case strings.HasPrefix(path, "/v1/upload/kanata"):
			return "/api/upload", cloneValues(query)
		case strings.HasPrefix(path, "/v1/upload/ryzumi"):
			return "/api/upload", cloneValues(query)
		default:
			return "/api/upload", cloneValues(query)
		}
	case ProviderKanata:
		switch {
		case path == "/v1/upload/kanata":
			return "/upload", cloneValues(query)
		case path == "/v1/upload":
			return "/upload", cloneValues(query)
		default:
			return "/upload", cloneValues(query)
		}
	case ProviderRyzumi:
		switch {
		case path == "/v1/upload/ryzumi":
			return "/api/uploader/ryzumicdn", cloneValues(query)
		case path == "/v1/upload":
			return "/api/uploader/ryzumicdn", cloneValues(query)
		default:
			return "/api/uploader/ryzumicdn", cloneValues(query)
		}
	default:
		return path, cloneValues(query)
	}
}

func toolPathTail(path, prefix string) string {
	if !strings.HasPrefix(path, strings.ToLower(prefix)) {
		return ""
	}
	return strings.TrimPrefix(strings.TrimPrefix(path, strings.ToLower(prefix)), "/")
}

func normalizePlate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ToUpper(strings.ReplaceAll(value, "-", " "))
	fields := strings.Fields(value)
	if len(fields) >= 3 {
		return strings.Join(fields[:3], " ")
	}
	if len(fields) == 1 {
		raw := fields[0]
		parts := make([]string, 0, 3)
		letters := func(i int) bool {
			return i < len(raw) && raw[i] >= 'A' && raw[i] <= 'Z'
		}
		i := 0
		for i < len(raw) && letters(i) {
			i++
		}
		if i > 0 {
			parts = append(parts, raw[:i])
		}
		j := i
		for j < len(raw) && raw[j] >= '0' && raw[j] <= '9' {
			j++
		}
		if j > i {
			parts = append(parts, raw[i:j])
		}
		if j < len(raw) {
			parts = append(parts, raw[j:])
		}
		if len(parts) >= 3 {
			return strings.Join(parts[:3], " ")
		}
	}
	return value
}

func searchChain(path string) []ProviderName {
	switch {
	case strings.Contains(path, "/google/image"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/google"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/youtube"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/spotify"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/pinterest"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/lyrics"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	case strings.Contains(path, "/tiktok"), strings.Contains(path, "/tidal"), strings.Contains(path, "/novel"):
		return []ProviderName{ProviderChocomilk, ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "/anime"), strings.Contains(path, "/manga"), strings.Contains(path, "/bstation"), strings.Contains(path, "/cookpad"), strings.Contains(path, "/wallpaper"), strings.Contains(path, "/pddikti"), strings.Contains(path, "/drama"), strings.Contains(path, "/film"), strings.Contains(path, "/cctv"), strings.Contains(path, "/dramabox"), strings.Contains(path, "/server-info"):
		if strings.Contains(path, "/wallpaper") {
			return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderKanata}
		}
		return []ProviderName{ProviderNexure, ProviderKanata}
	default:
		return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderKanata}
	}
}

func stalkChain(path string) []ProviderName {
	switch {
	case strings.Contains(path, "/instagram"):
		return []ProviderName{ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "/github"), strings.Contains(path, "/mobile-legends"), strings.Contains(path, "/free-fire"), strings.Contains(path, "/valorant"), strings.Contains(path, "/clash-of-clans"), strings.Contains(path, "/clash-royale"), strings.Contains(path, "/npm"):
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	default:
		return []ProviderName{ProviderRyzumi, ProviderNexure}
	}
}

func toolsChain(path string) []ProviderName {
	switch {
	case strings.Contains(path, "/translate"), strings.Contains(path, "/kbbi"), strings.Contains(path, "/ipinfo"), strings.Contains(path, "/carbon"):
		return []ProviderName{ProviderKanata, ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "/weather"), strings.Contains(path, "/cekresi"), strings.Contains(path, "/pln"), strings.Contains(path, "/pajak"), strings.Contains(path, "/qr"), strings.Contains(path, "/nsfw"), strings.Contains(path, "/brat"):
		return []ProviderName{ProviderNexure, ProviderKanata, ProviderRyzumi}
	case strings.Contains(path, "/whois"), strings.Contains(path, "/check-hosting"), strings.Contains(path, "/hargapangan"), strings.Contains(path, "/mc-lookup"), strings.Contains(path, "/qris-converter"), strings.Contains(path, "/turnstile"), strings.Contains(path, "/yt-transcript"), strings.Contains(path, "/shortlink/bypass"), strings.Contains(path, "/tinyurl"), strings.Contains(path, "/currency-converter"), strings.Contains(path, "/subdofinder"):
		return []ProviderName{ProviderRyzumi, ProviderNexure, ProviderYTDLP, ProviderKanata}
	case strings.Contains(path, "/isrc"):
		return []ProviderName{ProviderYTDLP, ProviderNexure, ProviderRyzumi}
	case strings.Contains(path, "/screenshot"), strings.Contains(path, "/kurs"), strings.Contains(path, "/gsmarena"), strings.Contains(path, "/distance"), strings.Contains(path, "/shorturl"), strings.Contains(path, "/listbank"), strings.Contains(path, "/cekbank"), strings.Contains(path, "/removebg"), strings.Contains(path, "/iphonechat"):
		return []ProviderName{ProviderYTDLP, ProviderNexure, ProviderRyzumi, ProviderKanata}
	default:
		return []ProviderName{ProviderKanata, ProviderNexure, ProviderRyzumi}
	}
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status == http.StatusNotFound || status == http.StatusMethodNotAllowed || status >= 500
}

func shouldRetryStatusForRoute(path string, status int) bool {
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "/v1/ai/image/") || strings.HasPrefix(lower, "/v1/ai/process/") || strings.HasPrefix(lower, "/v1/i2i/") {
		return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status == http.StatusBadRequest || status == http.StatusForbidden || status == http.StatusNotFound || status == http.StatusMethodNotAllowed || status >= 500
	}
	if strings.HasPrefix(lower, "/v1/download/") {
		return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status == http.StatusBadRequest || status == http.StatusForbidden || status == http.StatusNotFound || status == http.StatusMethodNotAllowed || status >= 500
	}
	if strings.HasPrefix(lower, "/v1/islamic/") {
		return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status == http.StatusBadRequest || status == http.StatusForbidden || status == http.StatusNotFound || status == http.StatusMethodNotAllowed || status >= 500
	}
	return shouldRetryStatus(status)
}

func copyProxyHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Host") || strings.EqualFold(key, "Content-Length") || strings.EqualFold(key, "Connection") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func aiProcessUpstreamPathAndQuery(slug string, provider ProviderName, query url.Values) (string, url.Values) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	normalized := strings.ReplaceAll(slug, "-", "")
	out := cloneValues(query)

	switch provider {
	case ProviderChocomilk:
		switch {
		case normalized == "tololi":
			return "/v1/i2i/tololi", buildAIProcessQuery(slug, out)
		case normalized == "enhance2x":
			return "/v1/i2i/enhance", buildAIProcessQuery(slug, out)
		case normalized == "nanobanana":
			return "/v1/i2i/nano-banana", buildAIProcessQuery(slug, out)
		default:
			return "/v1/i2i/" + slug, buildAIProcessQuery(slug, out)
		}
	case ProviderNexure:
		switch {
		case slug == "toanime":
			return "/api/ai/toanime", buildAIProcessQuery(slug, out)
		case slug == "removebg":
			return "/api/ai/removebg", buildAIProcessQuery(slug, out)
		case slug == "nsfw-check":
			return "/api/tools/nsfw-check", buildAIProcessQuery(slug, out)
		case slug == "upscale", slug == "enhance":
			return "/api/tools/upscale", buildAIProcessQuery(slug, out)
		case slug == "colorize":
			return "/api/ai/colorize", buildAIProcessQuery(slug, out)
		case slug == "faceswap":
			return "/api/ai/faceswap", buildAIProcessQuery(slug, out)
		case slug == "image2txt":
			return "/api/ai/image2txt", buildAIProcessQuery(slug, out)
		default:
			return "/api/ai/" + slug, buildAIProcessQuery(slug, out)
		}
	case ProviderRyzumi:
		switch {
		case slug == "toanime":
			return "/api/ai/toanime", buildAIProcessQuery(slug, out)
		case slug == "colorize":
			return "/api/ai/colorize", buildAIProcessQuery(slug, out)
		case slug == "faceswap":
			return "/api/ai/faceswap", buildAIProcessQuery(slug, out)
		case slug == "upscale":
			return "/api/ai/upscaler", buildAIProcessQuery(slug, out)
		case slug == "enhance":
			return "/api/ai/remini", buildAIProcessQuery(slug, out)
		case slug == "removebg":
			return "/api/ai/removebg", buildAIProcessQuery(slug, out)
		case slug == "waifu2x":
			return "/api/ai/waifu2x", buildAIProcessQuery(slug, out)
		case slug == "image2txt":
			return "/api/ai/image2txt", buildAIProcessQuery(slug, out)
		case slug == "nsfw-check":
			return "/api/tool/nsfw-checker", buildAIProcessQuery(slug, out)
		default:
			return "/api/ai/" + slug, buildAIProcessQuery(slug, out)
		}
	default:
		return "/v1/i2i/" + slug, buildAIProcessQuery(slug, out)
	}
}

func buildAIProcessQuery(slug string, query url.Values) url.Values {
	out := url.Values{}
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(slug)), "-", "")

	switch normalized {
	case "toanime":
		if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"), query.Get("imgUrl"))); u != "" {
			out.Set("url", u)
			out.Set("image_url", u)
			out.Set("imgUrl", u)
		}
		if style := strings.TrimSpace(firstNonEmpty(query.Get("style"), query.Get("mode"))); style != "" {
			out.Set("style", style)
		}
	case "colorize", "removebg", "waifu2x", "image2txt", "enhance", "enhance2x", "tololi", "nanobanana":
		if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"), query.Get("imgUrl"))); u != "" {
			out.Set("url", u)
			out.Set("image_url", u)
			out.Set("imgUrl", u)
		}
		if text := strings.TrimSpace(firstNonEmpty(query.Get("text"), query.Get("prompt"), query.Get("ask"))); text != "" {
			out.Set("text", text)
			out.Set("prompt", text)
		}
	case "faceswap":
		if original := strings.TrimSpace(firstNonEmpty(query.Get("original"), query.Get("url"), query.Get("image_url"))); original != "" {
			out.Set("original", original)
		}
		if face := strings.TrimSpace(firstNonEmpty(query.Get("face"), query.Get("face_url"), query.Get("faceUrl"))); face != "" {
			out.Set("face", face)
		}
	case "upscale":
		if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"), query.Get("imgUrl"))); u != "" {
			out.Set("url", u)
			out.Set("image_url", u)
			out.Set("imgUrl", u)
		}
		if scale := strings.TrimSpace(firstNonEmpty(query.Get("scale"), query.Get("factor"))); scale != "" {
			out.Set("scale", scale)
		}
	case "nsfwcheck":
		if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"), query.Get("imgUrl"))); u != "" {
			out.Set("url", u)
			out.Set("image_url", u)
			out.Set("imgUrl", u)
		}
	default:
		if u := strings.TrimSpace(firstNonEmpty(query.Get("url"), query.Get("image_url"), query.Get("imgUrl"))); u != "" {
			out.Set("url", u)
			out.Set("image_url", u)
			out.Set("imgUrl", u)
		}
	}

	for _, key := range []string{"model", "session", "style", "seed", "quality", "width", "height", "negative_prompt", "transparent", "aspectRatio", "enhance"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			out.Set(key, value)
		}
	}
	return out
}

func cloneURL(value *url.URL) *url.URL {
	if value == nil {
		return &url.URL{}
	}
	copy := *value
	return &copy
}

func joinURLPath(basePath, requestPath string) string {
	basePath = strings.TrimSuffix(basePath, "/")
	requestPath = "/" + strings.TrimPrefix(requestPath, "/")
	if basePath == "" || basePath == "/" {
		return requestPath
	}
	return basePath + requestPath
}

func (s *Service) cacheKey(method string, u *url.URL, body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("irag:%s:%s?%s:%s", strings.ToUpper(method), u.Path, u.RawQuery, hex.EncodeToString(sum[:]))
}

func (s *Service) writeCORS(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && s.isAllowedOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	} else if len(s.cfg.AllowedOrigins) == 0 {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Client-Id, X-Plan")
	w.Header().Set("Access-Control-Expose-Headers", "X-IRAG-Provider, X-IRAG-Latency-MS, X-IRAG-Cache")
}

func (s *Service) isAllowedOrigin(origin string) bool {
	if len(s.cfg.AllowedOrigins) == 0 {
		return true
	}
	for _, allowed := range s.cfg.AllowedOrigins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

func writeEnvelopeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func attemptedProvider(attempted []string) string {
	if len(attempted) == 0 {
		return ""
	}
	return attempted[len(attempted)-1]
}

func strconvInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func strconvBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
