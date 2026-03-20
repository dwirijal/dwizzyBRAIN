package providers

import (
	"fmt"
	"strings"
	"time"
)

func NewGeminiProvider(apiKey string) (*HTTPProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	return &HTTPProvider{
		name:    "gemini",
		client:  newJSONClient("https://generativelanguage.googleapis.com", 20*time.Second),
		path:    "/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey,
		headers: map[string]string{},
		buildBody: func(prompt string) any {
			return map[string]any{
				"contents": []map[string]any{
					{
						"parts": []map[string]string{
							{"text": prompt},
						},
					},
				},
			}
		},
		extract: extractGeminiText,
	}, nil
}
