package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const defaultInflightTTL = 30 * time.Second

type AgentRouter struct {
	providers   []LLMProvider
	locker      redis.Cmdable
	inflightTTL time.Duration
}

func NewAgentRouter(providers []LLMProvider, locker redis.Cmdable) *AgentRouter {
	return &AgentRouter{
		providers:   providers,
		locker:      locker,
		inflightTTL: defaultInflightTTL,
	}
}

func (r *AgentRouter) Ask(ctx context.Context, resourceID, prompt string) (string, error) {
	if len(r.providers) == 0 {
		return "", fmt.Errorf("at least one LLM provider is required")
	}
	if resourceID == "" {
		return "", fmt.Errorf("resourceID is required")
	}
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}

	lockKey := "inflight:ai:" + resourceID
	if err := r.acquireLock(ctx, lockKey); err != nil {
		return "", err
	}
	defer r.releaseLock(context.Background(), lockKey)

	var errs []error
	for _, provider := range r.providers {
		response, err := provider.Ask(ctx, prompt)
		if err == nil {
			return response, nil
		}

		errs = append(errs, fmt.Errorf("%s: %w", provider.Name(), err))
		if !IsRetryableProviderError(err) {
			return "", errors.Join(errs...)
		}
	}

	return "", errors.Join(errs...)
}

func (r *AgentRouter) acquireLock(ctx context.Context, key string) error {
	if r.locker == nil {
		return nil
	}

	locked, err := r.locker.SetNX(ctx, key, "1", r.inflightTTL).Result()
	if err != nil {
		return fmt.Errorf("acquire ai inflight lock %s: %w", key, err)
	}
	if !locked {
		return ErrDuplicateRequest
	}

	return nil
}

func (r *AgentRouter) releaseLock(ctx context.Context, key string) {
	if r.locker == nil {
		return
	}
	_ = r.locker.Del(ctx, key).Err()
}
