package providers

import (
	"fmt"
	"strings"
	"time"
)

func NewIRAGProvider(name, baseURL, path string) (*HTTPProvider, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("IRAG baseURL is required")
	}

	return &HTTPProvider{
		name:    name,
		client:  newJSONClient(baseURL, 15*time.Second),
		path:    path,
		headers: map[string]string{},
		buildBody: func(prompt string) any {
			return map[string]any{
				"prompt": prompt,
			}
		},
		extract: extractIRAGText,
	}, nil
}
