package archive

import (
	"fmt"
	"regexp"
	"strings"
)

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func RenderMarkdown(article Article) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: %d\n", article.ID))
	b.WriteString(fmt.Sprintf("title: %q\n", article.Title))
	b.WriteString(fmt.Sprintf("source: %q\n", article.Source))
	b.WriteString(fmt.Sprintf("source_url: %q\n", article.SourceURL))
	if article.FullURL != "" {
		b.WriteString(fmt.Sprintf("full_url: %q\n", article.FullURL))
	}
	b.WriteString(fmt.Sprintf("published_at: %q\n", article.PublishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")))
	b.WriteString(fmt.Sprintf("fetched_at: %q\n", article.FetchedAt.UTC().Format("2006-01-02T15:04:05Z07:00")))
	if article.Author != "" {
		b.WriteString(fmt.Sprintf("author: %q\n", article.Author))
	}
	if article.Metadata != nil {
		if article.Metadata.Category != "" {
			b.WriteString(fmt.Sprintf("category: %q\n", article.Metadata.Category))
		}
		if article.Metadata.Subcategory != "" {
			b.WriteString(fmt.Sprintf("subcategory: %q\n", article.Metadata.Subcategory))
		}
		if article.Metadata.Sentiment != "" {
			b.WriteString(fmt.Sprintf("sentiment: %q\n", article.Metadata.Sentiment))
		}
		if article.Metadata.ImportanceScore != nil {
			b.WriteString(fmt.Sprintf("importance_score: %.4f\n", *article.Metadata.ImportanceScore))
		}
	}
	if len(article.Entities) > 0 {
		b.WriteString("entities:\n")
		for _, entity := range article.Entities {
			label := entity.EntityName
			if label == "" {
				label = firstNonEmpty(entity.CoinID, entity.LlamaSlug, entity.EntityType)
			}
			b.WriteString(fmt.Sprintf("  - %q\n", label))
		}
	}
	b.WriteString("---\n\n")
	b.WriteString("# ")
	b.WriteString(article.Title)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("> Source: [%s](%s)\n", article.Source, article.SourceURL))
	if article.FullURL != "" {
		b.WriteString(fmt.Sprintf("> Canonical: [%s](%s)\n", article.FullURL, article.FullURL))
	}
	b.WriteString(fmt.Sprintf("> Published: %s\n", article.PublishedAt.UTC().Format("2006-01-02 15:04:05 MST")))
	if article.Author != "" {
		b.WriteString(fmt.Sprintf("> Author: %s\n", article.Author))
	}
	b.WriteString("\n")

	if article.Metadata != nil {
		if article.Metadata.SummaryShort != "" {
			b.WriteString("## Summary\n\n")
			b.WriteString(article.Metadata.SummaryShort)
			b.WriteString("\n\n")
		}
		if article.Metadata.KeyPoints != nil && len(article.Metadata.KeyPoints) > 0 {
			b.WriteString("## Key Points\n\n")
			for _, point := range article.Metadata.KeyPoints {
				point = strings.TrimSpace(point)
				if point == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(point)
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	if article.BodyPreview != "" {
		b.WriteString("## Preview\n\n")
		b.WriteString(article.BodyPreview)
		b.WriteString("\n\n")
	}

	if article.Metadata != nil && article.Metadata.SummaryLong != "" {
		b.WriteString("## AI Notes\n\n")
		b.WriteString(article.Metadata.SummaryLong)
		b.WriteString("\n\n")
	}

	if len(article.Entities) > 0 {
		b.WriteString("## Entities\n\n")
		for _, entity := range article.Entities {
			label := entity.EntityName
			if label == "" {
				label = firstNonEmpty(entity.CoinID, entity.LlamaSlug, entity.EntityType)
			}
			if label == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(label)
			if entity.IsPrimary {
				b.WriteString(" (primary)")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Metadata\n\n")
	b.WriteString(fmt.Sprintf("- article_id: `%d`\n", article.ID))
	b.WriteString(fmt.Sprintf("- external_id: `%s`\n", article.ExternalID))
	if article.Metadata != nil && article.Metadata.BreakingType != "" {
		b.WriteString(fmt.Sprintf("- breaking_type: `%s`\n", article.Metadata.BreakingType))
	}
	if article.Metadata != nil && article.Metadata.SentimentScore != nil {
		b.WriteString(fmt.Sprintf("- sentiment_score: `%.4f`\n", *article.Metadata.SentimentScore))
	}
	b.WriteString("\n")

	return b.String()
}

func Slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "article"
	}
	return value
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
