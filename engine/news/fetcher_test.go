package news

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRSSFetcherFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Fatalf("missing user-agent")
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer server.Close()

	fetcher := NewRSSFetcher()
	fetcher.client = server.Client()
	fetcher.now = func() time.Time { return time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC) }

	articles, err := fetcher.Fetch(context.Background(), Source{
		SourceName:  "coindesk",
		RSSURL:      sql.NullString{String: server.URL, Valid: true},
		FetchType:   "rss",
		DisplayName: "CoinDesk",
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("Fetch() len = %d, want 2", len(articles))
	}
	if articles[0].Title != "Bitcoin Rallies" {
		t.Fatalf("unexpected title = %q", articles[0].Title)
	}
	if !strings.Contains(articles[0].BodyPreview, "Bitcoin rose") {
		t.Fatalf("unexpected preview = %q", articles[0].BodyPreview)
	}
	if articles[0].Source != "coindesk" {
		t.Fatalf("unexpected source = %q", articles[0].Source)
	}
	if articles[0].PublishedAt.IsZero() {
		t.Fatalf("published_at should not be zero")
	}
}

func TestTelegramFetcherFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Fatalf("missing user-agent")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(sampleTelegramHTML))
	}))
	defer server.Close()

	fetcher := NewRSSFetcher()
	fetcher.client = server.Client()
	fetcher.now = func() time.Time { return time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC) }

	articles, err := fetcher.Fetch(context.Background(), Source{
		SourceName:  "beincrypto_id",
		RSSURL:      sql.NullString{String: server.URL, Valid: true},
		FetchType:   "telegram",
		DisplayName: "BeInCrypto Indonesia",
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("Fetch() len = %d, want 2", len(articles))
	}
	if articles[0].Title != "Skandal Binance: Tuduhan Pendanaan Terorisme US$1 Miliar" {
		t.Fatalf("unexpected title = %q", articles[0].Title)
	}
	if !strings.Contains(articles[0].BodyPreview, "Binance membantah") {
		t.Fatalf("unexpected preview = %q", articles[0].BodyPreview)
	}
	if articles[0].Source != "beincrypto_id" {
		t.Fatalf("unexpected source = %q", articles[0].Source)
	}
	if articles[0].PublishedAt.IsZero() {
		t.Fatalf("published_at should not be zero")
	}
	if !strings.Contains(articles[0].FullURL, "id.beincrypto.com") {
		t.Fatalf("unexpected full url = %q", articles[0].FullURL)
	}
}

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Sample Feed</title>
    <item>
      <guid>abc-123</guid>
      <title>Bitcoin Rallies</title>
      <link>https://example.com/bitcoin-rallies</link>
      <description><![CDATA[Bitcoin rose <b>5%</b> after ETF inflows.]]></description>
      <pubDate>Wed, 19 Mar 2026 00:00:00 GMT</pubDate>
      <author>Jane Doe</author>
    </item>
    <item>
      <title>Ethereum Update</title>
      <link>https://example.com/ethereum-update</link>
      <description><![CDATA[Ethereum gains on L2 demand.]]></description>
      <pubDate>Wed, 19 Mar 2026 00:10:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

const sampleTelegramHTML = `<!DOCTYPE html>
<html>
  <body>
    <div class="tgme_widget_message_wrap js-widget_message_wrap">
      <div class="tgme_widget_message text_not_supported_wrap js-widget_message" data-post="BeInCryptoIDNews/2925">
        <div class="tgme_widget_message_text js-message_text" dir="auto"><b>Skandal Binance: Tuduhan Pendanaan Terorisme US$1 Miliar</b><br/><br/>Gugatan US$1 miliar menuduh Binance memfasilitasi pendanaan terorisme. Binance membantah, tapi isu ini jadi sorotan global.<br/><br/><a href="https://id.beincrypto.com/gugatan-binance-grasi-trump-pendanaan-teror/" target="_blank" rel="noopener">BACA SELENGKAPNYA DI SINI</a></div>
        <div class="tgme_widget_message_footer compact js-message_footer">
          <div class="tgme_widget_message_info short js-message_info">
            <span class="tgme_widget_message_meta"><a class="tgme_widget_message_date" href="https://t.me/BeInCryptoIDNews/2925"><time datetime="2025-11-26T14:03:50+00:00" class="time">14:03</time></a></span>
          </div>
        </div>
      </div>
    </div>
    <div class="tgme_widget_message_wrap js-widget_message_wrap">
      <div class="tgme_widget_message text_not_supported_wrap js-widget_message" data-post="BeInCryptoIDNews/2930">
        <div class="tgme_widget_message_text js-message_text" dir="auto"><b>&#036;BTC</b><b>: Likuiditas Global Melonjak, Reli Tertunda ke 2026?</b><br/><br/>Meskipun 316 kali pemotongan suku bunga global terjadi, BTC sideways karena pemisahan dari likuiditas M2.<br/><br/><a href="https://id.beincrypto.com/pemotongan-suku-bunga-bank-sentral-likuiditas-bitcoin-2025/" target="_blank" rel="noopener">BACA SELENGKAPNYA DI SINI</a></div>
        <div class="tgme_widget_message_footer compact js-message_footer">
          <div class="tgme_widget_message_info short js-message_info">
            <span class="tgme_widget_message_meta"><a class="tgme_widget_message_date" href="https://t.me/BeInCryptoIDNews/2930"><time datetime="2025-11-28T12:10:24+00:00" class="time">12:10</time></a></span>
          </div>
        </div>
      </div>
    </div>
  </body>
</html>`
