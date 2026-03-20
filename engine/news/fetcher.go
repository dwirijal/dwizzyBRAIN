package news

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
)

const (
	defaultUserAgent = "Mozilla/5.0 (dwizzyBRAIN news fetcher)"
	defaultMaxItems  = 20
)

var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]*>`)

type Fetcher interface {
	Fetch(ctx context.Context, source Source) ([]Article, error)
}

type RSSFetcher struct {
	client    *http.Client
	userAgent string
	maxItems  int
	now       func() time.Time
}

func NewRSSFetcher() *RSSFetcher {
	return &RSSFetcher{
		client:    &http.Client{Timeout: 20 * time.Second},
		userAgent: defaultUserAgent,
		maxItems:  defaultMaxItems,
		now:       time.Now,
	}
}

func (f *RSSFetcher) Fetch(ctx context.Context, source Source) ([]Article, error) {
	fetchType := strings.ToLower(strings.TrimSpace(source.FetchType))
	if fetchType != "rss" && fetchType != "telegram" {
		return nil, fmt.Errorf("source %s fetch type %s is not supported", source.SourceName, source.FetchType)
	}
	rssURL := strings.TrimSpace(source.RSSURLValue())
	if rssURL == "" {
		return nil, fmt.Errorf("source %s missing rss_url", source.SourceName)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rssURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build rss request for %s: %w", source.SourceName, err)
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml;q=0.9, */*;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch rss %s: %w", source.SourceName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("fetch rss %s: http %d: %s", source.SourceName, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var articles []Article
	switch fetchType {
	case "rss":
		articles, err = parseRSS(source, resp.Body, f.now())
	case "telegram":
		articles, err = parseTelegram(source, resp.Body, f.now())
	default:
		return nil, fmt.Errorf("source %s fetch type %s not supported", source.SourceName, fetchType)
	}
	if err != nil {
		return nil, err
	}
	if len(articles) > f.maxItems {
		articles = articles[:f.maxItems]
	}
	return articles, nil
}

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	GUID        string `xml:"guid"`
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
	Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Author      string `xml:"author"`
	Enclosure   struct {
		URL string `xml:"url,attr"`
	} `xml:"enclosure"`
}

func parseRSS(source Source, reader io.Reader, now time.Time) ([]Article, error) {
	var feed rssFeed
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&feed); err != nil {
		return nil, fmt.Errorf("parse rss %s: %w", source.SourceName, err)
	}

	articles := make([]Article, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		title := strings.TrimSpace(html.UnescapeString(item.Title))
		link := strings.TrimSpace(item.Link)
		externalID := firstNonEmpty(strings.TrimSpace(item.GUID), link, title)
		if externalID == "" {
			externalID = hashString(source.SourceName + "|" + title + "|" + item.PubDate)
		}
		if link == "" {
			link = source.RSSURLValue() + "#" + externalID
		}

		bodyPreview := stripHTML(html.UnescapeString(item.Description))
		author := firstNonEmpty(strings.TrimSpace(item.Creator), strings.TrimSpace(item.Author))
		imageURL := strings.TrimSpace(item.Enclosure.URL)
		publishedAt := parseRSSDate(item.PubDate, now)

		articles = append(articles, Article{
			ExternalID:       externalID,
			Source:           source.SourceName,
			SourceURL:        link,
			Title:            title,
			BodyPreview:      bodyPreview,
			FullURL:          link,
			ImageURL:         imageURL,
			Author:           author,
			PublishedAt:      publishedAt,
			FetchedAt:        now,
			CPKind:           "",
			CPVotesPositive:  0,
			CPVotesNegative:  0,
			CPVotesImportant: 0,
		})
	}
	return articles, nil
}

func parseRSSDate(raw string, now time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return now.UTC()
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 MST",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC()
		}
	}
	return now.UTC()
}

func parseTelegram(source Source, reader io.Reader, now time.Time) ([]Article, error) {
	doc, err := xhtml.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("parse telegram %s: %w", source.SourceName, err)
	}

	blocks := findTelegramMessageBlocks(doc)
	articles := make([]Article, 0, len(blocks))
	for _, block := range blocks {
		postID := strings.TrimSpace(attrValue(block, "data-post"))
		textNode := findFirstNodeWithClass(block, "tgme_widget_message_text")
		if textNode == nil {
			continue
		}

		rawText := normalizeTelegramText(nodeText(textNode))
		if rawText == "" {
			continue
		}

		title := telegramTitle(rawText)
		fullURL := firstTelegramURL(block)
		if fullURL == "" {
			fullURL = firstNonEmpty(postTelegramURL(postID), source.RSSURLValue())
		}
		externalID := firstNonEmpty(postID, fullURL, hashString(source.SourceName+"|"+title+"|"+rawText))
		imageURL := firstTelegramImageURL(block)
		publishedAt := telegramPublishedAt(block, now)
		bodyPreview := telegramBodyPreview(rawText, title)
		author := telegramAuthor(block)

		articles = append(articles, Article{
			ExternalID:       externalID,
			Source:           source.SourceName,
			SourceURL:        fullURL,
			Title:            title,
			BodyPreview:      bodyPreview,
			FullURL:          fullURL,
			ImageURL:         imageURL,
			Author:           author,
			PublishedAt:      publishedAt,
			FetchedAt:        now,
			CPKind:           "",
			CPVotesPositive:  0,
			CPVotesNegative:  0,
			CPVotesImportant: 0,
		})
	}

	return articles, nil
}

func stripHTML(raw string) string {
	text := htmlTagPattern.ReplaceAllString(raw, " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	return strings.Join(strings.Fields(text), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func findTelegramMessageBlocks(root *xhtml.Node) []*xhtml.Node {
	blocks := make([]*xhtml.Node, 0)
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n == nil {
			return
		}
		if n.Type == xhtml.ElementNode && n.Data == "div" && hasClass(n, "tgme_widget_message_wrap") {
			if message := findFirstNodeWithClass(n, "tgme_widget_message"); message != nil {
				blocks = append(blocks, message)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return blocks
}

func findFirstNodeWithClass(root *xhtml.Node, className string) *xhtml.Node {
	var found *xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if found != nil || n == nil {
			return
		}
		if n.Type == xhtml.ElementNode && hasClass(n, className) {
			found = n
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
			if found != nil {
				return
			}
		}
	}
	walk(root)
	return found
}

func hasClass(node *xhtml.Node, className string) bool {
	for _, attr := range node.Attr {
		if attr.Key != "class" {
			continue
		}
		for _, part := range strings.Fields(attr.Val) {
			if part == className {
				return true
			}
		}
	}
	return false
}

func attrValue(node *xhtml.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func nodeText(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n == nil {
			return
		}
		switch n.Type {
		case xhtml.TextNode:
			b.WriteString(n.Data)
		case xhtml.ElementNode:
			if n.Data == "br" {
				b.WriteString("\n")
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
		default:
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
		}
	}
	walk(node)
	return b.String()
}

func normalizeTelegramText(raw string) string {
	raw = html.UnescapeString(raw)
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}

func telegramTitle(raw string) string {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func telegramBodyPreview(raw, title string) string {
	raw = strings.ReplaceAll(raw, "\n\n", " ")
	raw = strings.Join(strings.Fields(raw), " ")
	raw = strings.TrimSpace(raw)
	if title != "" {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, title))
		raw = strings.TrimSpace(strings.TrimPrefix(raw, ":"))
	}
	if len(raw) > 500 {
		raw = raw[:500]
	}
	return raw
}

func telegramPublishedAt(block *xhtml.Node, now time.Time) time.Time {
	if timeNode := findFirstTimeNode(block); timeNode != nil {
		if raw := attrValue(timeNode, "datetime"); raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				return parsed.UTC()
			}
		}
	}
	return now.UTC()
}

func findFirstTimeNode(root *xhtml.Node) *xhtml.Node {
	var found *xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if found != nil || n == nil {
			return
		}
		if n.Type == xhtml.ElementNode && n.Data == "time" {
			found = n
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
			if found != nil {
				return
			}
		}
	}
	walk(root)
	return found
}

func telegramAuthor(block *xhtml.Node) string {
	node := findFirstNodeWithClass(block, "tgme_widget_message_owner_name")
	if node == nil {
		return ""
	}
	return strings.TrimSpace(nodeText(node))
}

func firstTelegramURL(block *xhtml.Node) string {
	var links []string
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n == nil {
			return
		}
		if n.Type == xhtml.ElementNode && n.Data == "a" {
			if href := strings.TrimSpace(attrValue(n, "href")); href != "" {
				links = append(links, href)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(block)
	for _, link := range links {
		if isExternalTelegramLink(link) {
			return link
		}
	}
	return ""
}

func isExternalTelegramLink(link string) bool {
	link = strings.ToLower(strings.TrimSpace(link))
	if link == "" {
		return false
	}
	if strings.HasPrefix(link, "https://t.me/") || strings.HasPrefix(link, "http://t.me/") {
		return false
	}
	if strings.HasPrefix(link, "https://telegram.org/") || strings.HasPrefix(link, "http://telegram.org/") {
		return false
	}
	return strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")
}

func postTelegramURL(postID string) string {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return ""
	}
	return "https://t.me/" + strings.ReplaceAll(postID, ":", "/")
}

func firstTelegramImageURL(block *xhtml.Node) string {
	var links []string
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n == nil {
			return
		}
		if n.Type == xhtml.ElementNode && n.Data == "a" {
			if class := attrValue(n, "class"); strings.Contains(class, "tgme_widget_message_photo_wrap") || strings.Contains(class, "tgme_widget_message_video_player") {
				if href := strings.TrimSpace(attrValue(n, "style")); href != "" {
					if image := extractBackgroundImageURL(href); image != "" {
						links = append(links, image)
					}
				}
				for _, attr := range n.Attr {
					if attr.Key == "href" && strings.TrimSpace(attr.Val) != "" {
						links = append(links, strings.TrimSpace(attr.Val))
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(block)
	for _, link := range links {
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			return link
		}
	}
	return ""
}

func extractBackgroundImageURL(style string) string {
	idx := strings.Index(style, "background-image:url('")
	if idx < 0 {
		idx = strings.Index(style, "background-image:url(\"")
		if idx < 0 {
			return ""
		}
		start := idx + len("background-image:url(\"")
		end := strings.Index(style[start:], "\"")
		if end < 0 {
			return ""
		}
		return style[start : start+end]
	}
	start := idx + len("background-image:url('")
	end := strings.Index(style[start:], "'")
	if end < 0 {
		return ""
	}
	return style[start : start+end]
}

func hashString(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:16])
}
