package irag

import (
	"net/url"
	"strings"
)

func (s *Service) downloadUpstreamPathAndQuery(path string, provider ProviderName, query url.Values) (string, url.Values) {
	lower := strings.ToLower(strings.TrimSpace(path))
	slug := strings.TrimPrefix(lower, "/v1/download/")

	switch provider {
	case ProviderKanata:
		return kanataDownloadPathAndQuery(slug, query)
	case ProviderNexure:
		return nexureDownloadPathAndQuery(slug, query)
	case ProviderRyzumi:
		return ryzumiDownloadPathAndQuery(slug, query)
	case ProviderChocomilk:
		return chocomilkDownloadPathAndQuery(slug, query)
	case ProviderYTDLP:
		return ytdlpDownloadPathAndQuery(slug, query)
	default:
		return path, cloneValues(query)
	}
}

func kanataDownloadPathAndQuery(slug string, query url.Values) (string, url.Values) {
	out := buildDownloadURLQuery(query)
	switch {
	case slug == "youtube/info":
		return "/youtube/info", out
	case slug == "youtube/video":
		return "/youtube/download", setDownloadQuality(out, query, "720")
	case slug == "youtube/audio":
		return "/youtube/download-audio", out
	case strings.HasPrefix(slug, "youtube/playlist"), strings.HasPrefix(slug, "youtube/subtitle"):
		return "/youtube2/info", out
	case slug == "youtube/search":
		return "/youtube2/info", buildDownloadSearchQuery(query)
	case slug == "tiktok":
		return "/tiktok/fetch", out
	case slug == "tiktok/hd", slug == "douyin":
		return "/tiktok2", out
	case strings.HasPrefix(slug, "instagram"):
		return "/instagram/fetch", out
	case strings.HasPrefix(slug, "facebook"):
		return "/facebook/fetch", out
	case strings.HasPrefix(slug, "threads"):
		return "/threads/fetch", out
	case strings.HasPrefix(slug, "pinterest"):
		return "/pinterest/fetch", out
	case strings.HasPrefix(slug, "mediafire"):
		return "/mediafire/fetch", out
	case strings.HasPrefix(slug, "reddit"):
		return "/reddit/fetch", out
	default:
		return "/" + slug, out
	}
}

func nexureDownloadPathAndQuery(slug string, query url.Values) (string, url.Values) {
	out := buildDownloadURLQuery(query)
	switch {
	case slug == "aio":
		return "/api/download/aio", out
	case strings.HasPrefix(slug, "youtube/search"):
		return "/api/search/youtube", buildDownloadSearchQuery(query)
	case strings.HasPrefix(slug, "youtube/playlist"), strings.HasPrefix(slug, "youtube/subtitle"):
		return "/api/download/youtube", setDownloadQuality(out, query, "720")
	case strings.HasPrefix(slug, "youtube"):
		return "/api/download/youtube", setDownloadQuality(out, query, "720")
	case strings.HasPrefix(slug, "tiktok/hd"):
		return "/api/download/tiktok", setDownloadQuality(out, query, "hd")
	case strings.HasPrefix(slug, "tiktok"):
		return "/api/download/tiktok", out
	case strings.HasPrefix(slug, "instagram/story"):
		return "/api/download/ig-story", out
	case strings.HasPrefix(slug, "instagram"):
		return "/api/download/instagram", out
	case strings.HasPrefix(slug, "spotify/playlist"):
		return "/api/download/spotify", out
	case strings.HasPrefix(slug, "spotify"):
		return "/api/download/spotify", out
	case strings.HasPrefix(slug, "facebook"):
		return "/api/download/facebook", out
	case strings.HasPrefix(slug, "twitter"):
		return "/api/download/twitter", out
	case strings.HasPrefix(slug, "threads"):
		return "/api/download/threads", out
	case strings.HasPrefix(slug, "pinterest"):
		return "/api/download/pinterest", out
	case strings.HasPrefix(slug, "soundcloud"):
		return "/api/download/soundcloud", out
	case strings.HasPrefix(slug, "gdrive"):
		return "/api/download/gdrive", out
	case strings.HasPrefix(slug, "bilibili"), strings.HasPrefix(slug, "bstation"):
		return "/api/download/bstation", out
	case strings.HasPrefix(slug, "scribd"):
		return "/api/download/scribd", out
	case strings.HasPrefix(slug, "mediafire"):
		return "/api/download/mediafire", out
	case strings.HasPrefix(slug, "mega"):
		return "/api/download/mega", out
	case strings.HasPrefix(slug, "terabox"):
		return "/api/download/terabox", out
	case strings.HasPrefix(slug, "pixeldrain"):
		return "/api/download/pixeldrain", out
	case strings.HasPrefix(slug, "krakenfiles"):
		return "/api/download/kfiles", out
	case strings.HasPrefix(slug, "danbooru"):
		return "/api/download/danbooru", out
	case strings.HasPrefix(slug, "reddit"):
		return "/api/download/reddit", out
	default:
		return "/api/download/" + slug, out
	}
}

func ryzumiDownloadPathAndQuery(slug string, query url.Values) (string, url.Values) {
	out := buildDownloadURLQuery(query)
	switch {
	case slug == "aio":
		return "/api/downloader/all-in-one", out
	case strings.HasPrefix(slug, "youtube/search"):
		return "/api/search/yt", buildDownloadSearchQuery(query)
	case strings.HasPrefix(slug, "youtube/playlist"), strings.HasPrefix(slug, "youtube/subtitle"):
		return "/api/downloader/ytmp4", setDownloadQuality(out, query, "720p")
	case strings.HasPrefix(slug, "youtube/audio"):
		return "/api/downloader/ytmp3", out
	case strings.HasPrefix(slug, "youtube"):
		return "/api/downloader/ytmp4", setDownloadQuality(out, query, "720p")
	case strings.HasPrefix(slug, "tiktok/hd"), strings.HasPrefix(slug, "douyin"):
		return "/api/downloader/v2/ttdl", out
	case strings.HasPrefix(slug, "tiktok"):
		return "/api/downloader/ttdl", out
	case strings.HasPrefix(slug, "instagram"):
		return "/api/downloader/igdl", out
	case strings.HasPrefix(slug, "facebook"):
		return "/api/downloader/fbdl", out
	case strings.HasPrefix(slug, "twitter"):
		return "/api/downloader/twitter", out
	case strings.HasPrefix(slug, "threads"):
		return "/api/downloader/threads", out
	case strings.HasPrefix(slug, "pinterest"):
		return "/api/downloader/pinterest", out
	case strings.HasPrefix(slug, "soundcloud"):
		return "/api/downloader/soundcloud", out
	case strings.HasPrefix(slug, "gdrive"):
		return "/api/downloader/gdrive", out
	case strings.HasPrefix(slug, "bilibili"), strings.HasPrefix(slug, "bstation"):
		return "/api/downloader/bilibili", out
	case strings.HasPrefix(slug, "mega"):
		return "/api/downloader/mega", out
	case strings.HasPrefix(slug, "terabox"):
		return "/api/downloader/terabox", out
	case strings.HasPrefix(slug, "pixeldrain"):
		return "/api/downloader/pixeldrain", out
	case strings.HasPrefix(slug, "krakenfiles"):
		return "/api/downloader/kfiles", out
	case strings.HasPrefix(slug, "danbooru"):
		return "/api/downloader/danbooru", out
	case strings.HasPrefix(slug, "mediafire"):
		return "/api/downloader/mediafire", out
	case strings.HasPrefix(slug, "reddit"):
		return "/api/downloader/all-in-one", out
	default:
		return "/api/downloader/" + slug, out
	}
}

func chocomilkDownloadPathAndQuery(slug string, query url.Values) (string, url.Values) {
	out := buildDownloadURLQuery(query)
	switch {
	case strings.HasPrefix(slug, "twitter"):
		return "/v1/download/twitter", out
	case strings.HasPrefix(slug, "threads"):
		return "/v1/download/threads", out
	case strings.HasPrefix(slug, "facebook"):
		return "/v1/download/facebook", out
	case strings.HasPrefix(slug, "tidal"):
		return "/v1/download/tidal", out
	case strings.HasPrefix(slug, "deezer"):
		return "/v1/download/deezer", out
	case strings.HasPrefix(slug, "capcut"):
		return "/v1/download/capcut", out
	default:
		return "/v1/download/" + slug, out
	}
}

func ytdlpDownloadPathAndQuery(slug string, query url.Values) (string, url.Values) {
	out := buildDownloadURLQuery(query)
	switch {
	case strings.HasPrefix(slug, "youtube/info"):
		return "/info/", out
	case strings.HasPrefix(slug, "youtube/video"):
		return "/download/", setDownloadQuality(out, query, firstNonEmpty(strings.TrimSpace(query.Get("quality")), "720"))
	case strings.HasPrefix(slug, "youtube/audio"):
		return "/download/audio", out
	case strings.HasPrefix(slug, "youtube/playlist"):
		return "/download/playlist", out
	case strings.HasPrefix(slug, "youtube/subtitle"):
		return "/download/ytsub", out
	case strings.HasPrefix(slug, "tiktok/hd"), strings.HasPrefix(slug, "douyin"):
		return "/downloader/tiktokhd", out
	case strings.HasPrefix(slug, "spotify/playlist"):
		return "/spotify/download/playlist", out
	case strings.HasPrefix(slug, "spotify"):
		return "/spotify/download/audio", out
	case strings.HasPrefix(slug, "mediafire"):
		return "/downloader/mediafire", out
	case strings.HasPrefix(slug, "soundcloud/playlist"):
		return "/downloader/soundcloud/playlist", out
	case strings.HasPrefix(slug, "soundcloud"):
		return "/downloader/soundcloud", out
	case strings.HasPrefix(slug, "videy"):
		return "/downloader/videy", out
	case strings.HasPrefix(slug, "sfile"):
		return "/sfile", out
	case strings.HasPrefix(slug, "shopee/video"):
		return "/shopee/video", out
	case strings.HasPrefix(slug, "nhentai"):
		return "/nhentai", out
	case strings.HasPrefix(slug, "applemusic"):
		return "/download/applemusic", out
	default:
		return "/download/" + slug, out
	}
}

func buildDownloadURLQuery(query url.Values) url.Values {
	out := cloneValues(query)
	if value := downloadURLValue(query); value != "" {
		out.Set("url", value)
	}
	if quality := strings.TrimSpace(firstNonEmpty(query.Get("quality"), query.Get("format"))); quality != "" {
		out.Set("quality", quality)
		out.Set("format", quality)
	}
	if lang := strings.TrimSpace(query.Get("lang")); lang != "" {
		out.Set("lang", lang)
	}
	if model := strings.TrimSpace(query.Get("model")); model != "" {
		out.Set("model", model)
	}
	if session := strings.TrimSpace(query.Get("session")); session != "" {
		out.Set("session", session)
	}
	return out
}

func buildDownloadSearchQuery(query url.Values) url.Values {
	out := cloneValues(query)
	search := downloadSearchValue(query)
	if search != "" {
		out.Set("q", search)
		out.Set("query", search)
	}
	return out
}

func setDownloadQuality(query url.Values, raw url.Values, fallback string) url.Values {
	quality := strings.TrimSpace(firstNonEmpty(raw.Get("quality"), raw.Get("format"), fallback))
	if quality == "" {
		return query
	}
	query.Set("quality", quality)
	query.Set("format", quality)
	return query
}

func downloadURLValue(query url.Values) string {
	for _, key := range []string{"url", "link", "target", "source"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func downloadSearchValue(query url.Values) string {
	for _, key := range []string{"q", "query", "keyword", "search"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
	}
	return downloadURLValue(query)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func downloadRouteAllowed(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	switch {
	case lower == "/v1/download/aio",
		lower == "/v1/download/youtube",
		lower == "/v1/download/youtube/info",
		lower == "/v1/download/youtube/video",
		lower == "/v1/download/youtube/audio",
		lower == "/v1/download/youtube/playlist",
		lower == "/v1/download/youtube/subtitle",
		lower == "/v1/download/youtube/search",
		lower == "/v1/download/tiktok",
		lower == "/v1/download/tiktok/hd",
		lower == "/v1/download/douyin",
		lower == "/v1/download/instagram",
		lower == "/v1/download/instagram/story",
		lower == "/v1/download/spotify",
		lower == "/v1/download/spotify/playlist",
		lower == "/v1/download/facebook",
		lower == "/v1/download/threads",
		lower == "/v1/download/twitter",
		lower == "/v1/download/pinterest",
		lower == "/v1/download/soundcloud",
		lower == "/v1/download/soundcloud/playlist",
		lower == "/v1/download/gdrive",
		lower == "/v1/download/bilibili",
		lower == "/v1/download/bstation",
		lower == "/v1/download/tidal",
		lower == "/v1/download/deezer",
		lower == "/v1/download/capcut",
		lower == "/v1/download/scribd",
		lower == "/v1/download/mediafire",
		lower == "/v1/download/mega",
		lower == "/v1/download/terabox",
		lower == "/v1/download/pixeldrain",
		lower == "/v1/download/krakenfiles",
		lower == "/v1/download/danbooru",
		lower == "/v1/download/reddit",
		lower == "/v1/download/applemusic",
		lower == "/v1/download/videy",
		lower == "/v1/download/sfile",
		lower == "/v1/download/shopee/video",
		lower == "/v1/download/nhentai":
		return true
	case strings.HasPrefix(lower, "/v1/download/youtube/"),
		strings.HasPrefix(lower, "/v1/download/tiktok/"),
		strings.HasPrefix(lower, "/v1/download/instagram/"),
		strings.HasPrefix(lower, "/v1/download/spotify/"),
		strings.HasPrefix(lower, "/v1/download/soundcloud/"),
		strings.HasPrefix(lower, "/v1/download/bilibili/"),
		strings.HasPrefix(lower, "/v1/download/bstation/"):
		return true
	case strings.HasPrefix(lower, "/v1/download/facebook/"),
		strings.HasPrefix(lower, "/v1/download/threads/"),
		strings.HasPrefix(lower, "/v1/download/twitter/"),
		strings.HasPrefix(lower, "/v1/download/pinterest/"),
		strings.HasPrefix(lower, "/v1/download/gdrive/"),
		strings.HasPrefix(lower, "/v1/download/tidal/"),
		strings.HasPrefix(lower, "/v1/download/deezer/"),
		strings.HasPrefix(lower, "/v1/download/capcut/"),
		strings.HasPrefix(lower, "/v1/download/scribd/"),
		strings.HasPrefix(lower, "/v1/download/mediafire/"),
		strings.HasPrefix(lower, "/v1/download/mega/"),
		strings.HasPrefix(lower, "/v1/download/terabox/"),
		strings.HasPrefix(lower, "/v1/download/pixeldrain/"),
		strings.HasPrefix(lower, "/v1/download/krakenfiles/"),
		strings.HasPrefix(lower, "/v1/download/danbooru/"),
		strings.HasPrefix(lower, "/v1/download/reddit/"),
		strings.HasPrefix(lower, "/v1/download/applemusic/"),
		strings.HasPrefix(lower, "/v1/download/videy/"),
		strings.HasPrefix(lower, "/v1/download/sfile/"),
		strings.HasPrefix(lower, "/v1/download/nhentai/"):
		return true
	default:
		return false
	}
}
