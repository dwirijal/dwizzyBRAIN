package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

var ErrDuplicateRequest = errors.New("duplicate inflight ai request")

type LLMProvider interface {
	Name() string
	Ask(ctx context.Context, prompt string) (string, error)
}

type RetryableProviderError struct {
	Provider string
	Status   int
	Err      error
}

func (e *RetryableProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.Status > 0 {
		return fmt.Sprintf("%s retryable failure (%d): %v", e.Provider, e.Status, e.Err)
	}
	return fmt.Sprintf("%s retryable failure: %v", e.Provider, e.Err)
}

func (e *RetryableProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsRetryableProviderError(err error) bool {
	var target *RetryableProviderError
	return errors.As(err, &target)
}

func NewRetryableProviderError(provider string, status int, err error) error {
	return &RetryableProviderError{
		Provider: provider,
		Status:   status,
		Err:      err,
	}
}

func IsRetryableHTTPStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status >= http.StatusInternalServerError
}
