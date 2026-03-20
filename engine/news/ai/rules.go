package ai

import (
	"math"
	"regexp"
	"strings"
)

var (
	wordBoundary  = regexp.MustCompile(`\s+`)
	nonWord       = regexp.MustCompile(`[^a-z0-9]+`)
	splitSentence = regexp.MustCompile(`[.!?]+`)
)

var positiveWords = []string{
	"rally", "surge", "soar", "jump", "gain", "bullish", "upgrade", "approve", "approval",
	"inflow", "record", "adoption", "partnership", "launch", "integrate", "grows", "growth",
	"buy", "wins", "expands", "milestone", "beats",
}

var negativeWords = []string{
	"hack", "exploit", "breach", "lawsuit", "probe", "investigation", "regulation", "delist",
	"delisting", "ban", "crash", "fall", "drop", "liquidation", "outage", "risk", "fraud",
	"exploiters", "drain", "drained",
}

type analyzedArticle struct {
	metadata Metadata
	entities []Entity
}

func analyzeArticle(a Article, coins []CoinEntity, protocols []ProtocolEntity) analyzedArticle {
	text := normalizedText(a.Title + " " + a.BodyPreview)

	sentimentScore := sentimentScore(text)
	sentiment := sentimentLabel(sentimentScore)
	category := detectCategory(text)
	breaking, breakingType := detectBreaking(text, category)

	coinHits := matchCoinEntities(text, a.Title, coins)
	protocolHits := matchProtocolEntities(text, a.Title, protocols)
	entities := make([]Entity, 0, len(coinHits)+len(protocolHits))
	entities = append(entities, coinHits...)
	entities = append(entities, protocolHits...)

	summaryShort := buildSummaryShort(a.Title, a.BodyPreview)
	summaryLong := buildSummaryLong(a.Title, a.BodyPreview)
	keyPoints := buildKeyPoints(a.Title, a.BodyPreview)

	importance := importanceScore(a.SourceCredibility, sentimentScore, breaking, len(entities), a.CPVotesPositive, a.CPVotesImportant)
	if importance > 100 {
		importance = 100
	}
	if importance < 0 {
		importance = 0
	}

	return analyzedArticle{
		metadata: Metadata{
			ArticleID:           a.ID,
			SummaryShort:        summaryShort,
			SummaryLong:         summaryLong,
			KeyPoints:           keyPoints,
			Sentiment:           sentiment,
			SentimentScore:      round3(sentimentScore),
			Category:            category,
			Subcategory:         "",
			ImportanceScore:     round3(importance),
			IsBreaking:          breaking,
			BreakingType:        breakingType,
			ModelUsed:           "heuristic/rss-v1",
			ProcessingLatencyMS: 0,
		},
		entities: entities,
	}
}

func normalizedText(raw string) string {
	raw = strings.ToLower(raw)
	raw = nonWord.ReplaceAllString(raw, " ")
	raw = wordBoundary.ReplaceAllString(raw, " ")
	return strings.TrimSpace(raw)
}

func sentimentScore(text string) float64 {
	score := 0.0
	for _, word := range positiveWords {
		if strings.Contains(text, word) {
			score += 0.15
		}
	}
	for _, word := range negativeWords {
		if strings.Contains(text, word) {
			score -= 0.18
		}
	}
	if score > 1 {
		score = 1
	}
	if score < -1 {
		score = -1
	}
	return score
}

func sentimentLabel(score float64) string {
	switch {
	case score >= 0.25:
		return "bullish"
	case score <= -0.25:
		return "bearish"
	case math.Abs(score) < 0.15:
		return "neutral"
	default:
		return "mixed"
	}
}

func detectCategory(text string) string {
	type rule struct {
		category string
		terms    []string
	}
	rules := []rule{
		{"hack_exploit", []string{"hack", "exploit", "breach", "drain", "drained", "stolen", "exploiters"}},
		{"regulation", []string{"sec", "regulation", "lawsuit", "court", "investigation", "probe", "compliance", "approves", "approval"}},
		{"partnership", []string{"partnership", "partner", "collaboration", "integrates", "integrated"}},
		{"listing_delisting", []string{"listing", "listed", "delisting", "delisted", "exchange listing"}},
		{"fundraising", []string{"fundraise", "fundraising", "raises", "raised", "seed round", "series a", "series b"}},
		{"whale_movement", []string{"whale", "whales", "large transfer", "transfer"}},
		{"layer2", []string{"layer 2", "l2", "rollup", "zk", "optimistic"}},
		{"defi", []string{"defi", "dex", "liquidity", "yield", "pool"}},
		{"adoption", []string{"adoption", "users", "active users", "merchant", "payment", "payments"}},
		{"macro", []string{"macro", "fed", "inflation", "rates", "cpi", "jobs"}},
		{"market_analysis", []string{"analysis", "chart", "price analysis", "market analysis"}},
		{"technology", []string{"upgrade", "mainnet", "protocol", "testnet", "upgrade"}},
		{"nft", []string{"nft", "nfts", "collectible"}},
	}
	for _, rule := range rules {
		for _, term := range rule.terms {
			if strings.Contains(text, term) {
				return rule.category
			}
		}
	}
	return "other"
}

func detectBreaking(text, category string) (bool, string) {
	if category == "hack_exploit" || category == "regulation" {
		return true, category
	}
	if strings.Contains(text, "breaking") || strings.Contains(text, "urgent") || strings.Contains(text, "alert") {
		return true, "breaking_news"
	}
	return false, ""
}

func matchCoinEntities(text, title string, coins []CoinEntity) []Entity {
	normalized := " " + normalizedText(text) + " "
	titleNorm := " " + normalizedText(title) + " "
	seen := make(map[string]struct{})
	entities := make([]Entity, 0)

	for _, coin := range coins {
		terms := coinTerms(coin)
		hits := 0
		for _, term := range terms {
			needle := " " + normalizedText(term) + " "
			if strings.TrimSpace(needle) == "" {
				continue
			}
			if len(strings.TrimSpace(term)) < 3 && !strings.EqualFold(term, "btc") && !strings.EqualFold(term, "eth") {
				continue
			}
			if strings.Contains(normalized, needle) {
				hits++
			} else if strings.Contains(titleNorm, needle) {
				hits++
			}
		}
		if hits == 0 {
			continue
		}
		if _, ok := seen[coin.CoinID]; ok {
			continue
		}
		seen[coin.CoinID] = struct{}{}
		entities = append(entities, Entity{
			CoinID:       coin.CoinID,
			EntityType:   "coin",
			EntityName:   firstNonEmpty(coin.Name, coin.Symbol, coin.CoinID),
			IsPrimary:    len(entities) == 0,
			MentionCount: hits,
			Confidence:   confidenceFromHits(hits),
		})
	}
	return entities
}

func matchProtocolEntities(text, title string, protocols []ProtocolEntity) []Entity {
	normalized := " " + normalizedText(text) + " "
	titleNorm := " " + normalizedText(title) + " "
	seen := make(map[string]struct{})
	entities := make([]Entity, 0)

	for _, protocol := range protocols {
		terms := protocolTerms(protocol)
		hits := 0
		for _, term := range terms {
			needle := " " + normalizedText(term) + " "
			if strings.TrimSpace(needle) == "" {
				continue
			}
			if strings.Contains(normalized, needle) || strings.Contains(titleNorm, needle) {
				hits++
			}
		}
		if hits == 0 {
			continue
		}
		if _, ok := seen[protocol.Slug]; ok {
			continue
		}
		seen[protocol.Slug] = struct{}{}
		entities = append(entities, Entity{
			LlamaSlug:    protocol.Slug,
			EntityType:   "protocol",
			EntityName:   firstNonEmpty(protocol.Name, protocol.Slug),
			IsPrimary:    len(entities) == 0,
			MentionCount: hits,
			Confidence:   confidenceFromHits(hits),
		})
	}
	return entities
}

func coinTerms(coin CoinEntity) []string {
	terms := []string{coin.CoinID, coin.Symbol, coin.Name}
	return dedupeStrings(terms)
}

func protocolTerms(protocol ProtocolEntity) []string {
	terms := []string{protocol.Slug, protocol.Name}
	return dedupeStrings(terms)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func buildSummaryShort(title, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return strings.TrimSpace(title)
	}
	sentences := splitSentence.Split(body, 2)
	lead := ""
	if len(sentences) > 0 {
		lead = strings.TrimSpace(sentences[0])
	}
	if lead == "" {
		return strings.TrimSpace(title)
	}
	return strings.TrimSpace(title + ": " + lead)
}

func buildSummaryLong(title, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if body == "" {
		return title
	}
	if title == "" {
		return body
	}
	return title + ". " + body
}

func buildKeyPoints(title, body string) []string {
	points := make([]string, 0, 3)
	title = strings.TrimSpace(title)
	if title != "" {
		points = append(points, title)
	}
	for _, sentence := range splitSentence.Split(body, -1) {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		points = append(points, sentence)
		if len(points) >= 3 {
			break
		}
	}
	if len(points) == 0 {
		points = append(points, "News article processed from RSS feed")
	}
	return points
}

func confidenceFromHits(hits int) float64 {
	switch {
	case hits >= 3:
		return 0.95
	case hits == 2:
		return 0.85
	case hits == 1:
		return 0.70
	default:
		return 0.50
	}
}

func importanceScore(credibility, sentiment float64, breaking bool, entityCount int, votesPositive, votesImportant int) float64 {
	score := credibility * 40
	score += math.Abs(sentiment) * 20
	score += float64(entityCount) * 6
	score += float64(votesPositive) * 0.6
	score += float64(votesImportant) * 1.5
	if breaking {
		score += 15
	}
	return score
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
