package providers

import (
	"fmt"
	"strings"
	"time"
)

func NewGroqProvider(apiKey string) (*HTTPProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("GROQ_API_KEY is required")
	}

	return &HTTPProvider{
		name:   "groq",
		client: newJSONClient("https://api.groq.com/openai/v1", 20*time.Second),
		path:   "/chat/completions",
		headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
		buildBody: func(prompt string) any {
			return map[string]any{
				"model": "llama-3.1-8b-instant",
				"messages": []map[string]string{
					{"role": "user", "content": prompt},
				},
			}
		},
		extract: extractGroqText,
	}, nil
}
