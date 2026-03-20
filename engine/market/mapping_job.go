package market

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

const defaultMappingSyncInterval = 6 * time.Hour

type mappingSyncer interface {
	SyncAll(ctx context.Context, exchanges []string) ([]MappingSyncResult, error)
}

type MappingSyncJob struct {
	service   mappingSyncer
	exchanges []string
	interval  time.Duration
	logger    *log.Logger
}

func NewMappingSyncJob(service mappingSyncer, exchanges []string, interval time.Duration, logger *log.Logger) *MappingSyncJob {
	if interval <= 0 {
		interval = defaultMappingSyncInterval
	}

	normalized := make([]string, 0, len(exchanges))
	for _, exchange := range exchanges {
		exchange = strings.ToLower(strings.TrimSpace(exchange))
		if exchange == "" {
			continue
		}
		normalized = append(normalized, exchange)
	}

	return &MappingSyncJob{
		service:   service,
		exchanges: normalized,
		interval:  interval,
		logger:    logger,
	}
}

func (j *MappingSyncJob) RunOnce(ctx context.Context) ([]MappingSyncResult, error) {
	if j.service == nil {
		return nil, fmt.Errorf("mapping sync service is required")
	}
	if len(j.exchanges) == 0 {
		return nil, fmt.Errorf("at least one exchange is required")
	}

	results, err := j.service.SyncAll(ctx, j.exchanges)
	if err != nil {
		return nil, err
	}

	if j.logger != nil {
		for _, result := range results {
			j.logger.Printf(
				"mapping sync exchange=%s matched=%d skipped=%d unmatched=%d validated=%d active=%d delisted=%d",
				result.Exchange,
				result.Build.Matched,
				result.Build.Skipped,
				result.Build.Unmatched,
				result.Validation.Validated,
				result.Validation.Active,
				result.Validation.Delisted,
			)
		}
	}

	return results, nil
}

func (j *MappingSyncJob) Run(ctx context.Context) error {
	if _, err := j.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := j.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}
