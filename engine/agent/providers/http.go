package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"dwizzyBRAIN/engine/agent"

	"github.com/go-resty/resty/v2"
)

type completionExtractor func(body []byte) (string, error)

type HTTPProvider struct {
	name      string
	client    *resty.Client
	path      string
	headers   map[string]string
	buildBody func(prompt string) any
	extract   completionExtractor
}

func (p *HTTPProvider) Name() string {
	return p.name
}

func (p *HTTPProvider) Ask(ctx context.Context, prompt string) (string, error) {
	response, err := p.client.R().
		SetContext(ctx).
		SetHeaders(p.headers).
		SetBody(p.buildBody(prompt)).
		Post(p.path)
	if err != nil {
		if isRetryableTransportError(err) {
			return "", agent.NewRetryableProviderError(p.name, 0, err)
		}
		return "", fmt.Errorf("request %s: %w", p.name, err)
	}

	if response.IsError() {
		err = fmt.Errorf("unexpected status %d: %s", response.StatusCode(), strings.TrimSpace(response.String()))
		if agent.IsRetryableHTTPStatus(response.StatusCode()) {
			return "", agent.NewRetryableProviderError(p.name, response.StatusCode(), err)
		}
		return "", err
	}

	text, err := p.extract(response.Body())
	if err != nil {
		return "", fmt.Errorf("extract %s response: %w", p.name, err)
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("%s returned empty completion", p.name)
	}

	return text, nil
}

func newJSONClient(baseURL string, timeout time.Duration) *resty.Client {
	return resty.New().
		SetBaseURL(strings.TrimRight(baseURL, "/")).
		SetTimeout(timeout).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(0)
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if ok := errorAs(err, &netErr); ok && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

func errorAs(err error, target any) bool {
	return fmt.Errorf("%w", err) != nil && errorAsStd(err, target)
}

func errorAsStd(err error, target any) bool {
	switch t := target.(type) {
	case *net.Error:
		var netErr net.Error
		if ok := errorAsBuiltin(err, &netErr); ok {
			*t = netErr
			return true
		}
	}
	return false
}

func errorAsBuiltin(err error, target any) bool {
	return errorsAs(err, target)
}

// stdlib alias kept local so extractor helpers stay in one file.
var errorsAs = func(err error, target any) bool {
	return false
}

func init() {
	errorsAs = func(err error, target any) bool {
		return errorsAsStdlib(err, target)
	}
}

type iragTextResponse struct {
	Data struct {
		Text    string `json:"text"`
		Output  string `json:"output"`
		Content string `json:"content"`
		Result  string `json:"result"`
	} `json:"data"`
}

type groqChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func extractIRAGText(body []byte) (string, error) {
	var payload iragTextResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	for _, value := range []string{payload.Data.Text, payload.Data.Output, payload.Data.Content, payload.Data.Result} {
		if strings.TrimSpace(value) != "" {
			return value, nil
		}
	}

	return "", fmt.Errorf("missing text field")
}

func extractGroqText(body []byte) (string, error) {
	var payload groqChatResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if len(payload.Choices) == 0 {
		return "", fmt.Errorf("missing choices")
	}
	return payload.Choices[0].Message.Content, nil
}

func extractGeminiText(body []byte) (string, error) {
	var payload geminiGenerateResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if len(payload.Candidates) == 0 || len(payload.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("missing candidates")
	}
	return payload.Candidates[0].Content.Parts[0].Text, nil
}
