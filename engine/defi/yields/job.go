package yields

import (
	"context"
	"fmt"
	"log"
	"time"
)

const defaultSyncInterval = 1 * time.Hour

type Job struct {
	service  *Service
	interval time.Duration
	logger   *log.Logger
}

func NewJob(service *Service, interval time.Duration, logger *log.Logger) *Job {
	if interval <= 0 {
		interval = defaultSyncInterval
	}
	return &Job{
		service:  service,
		interval: interval,
		logger:   logger,
	}
}

func (j *Job) RunOnce(ctx context.Context) (Result, error) {
	if j.service == nil {
		return Result{}, fmt.Errorf("yields service is required")
	}

	result, err := j.service.RunOnce(ctx)
	if err != nil {
		return Result{}, err
	}
	if j.logger != nil {
		j.logger.Printf("yields sync pools=%d upserted=%d backfilled=%d", result.PoolsFetched, result.PoolsUpserted, result.PoolsBackfilled)
	}
	return result, nil
}

func (j *Job) Run(ctx context.Context) error {
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
