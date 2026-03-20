package news

import (
	"context"
	"log"
	"time"
)

type Job struct {
	service  *Service
	interval time.Duration
	logger   *log.Logger
}

func NewJob(service *Service, interval time.Duration, logger *log.Logger) *Job {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &Job{
		service:  service,
		interval: interval,
		logger:   logger,
	}
}

func (j *Job) RunOnce(ctx context.Context) (Result, error) {
	result, err := j.service.RunOnce(ctx)
	if j.logger != nil {
		if len(result.FailedSources) > 0 {
			j.logger.Printf("news sync processed=%d fetched=%d inserted=%d failures=%d failed_sources=%v",
				result.SourcesProcessed, result.ArticlesFetched, result.ArticlesInserted, result.Failures, result.FailedSources)
		} else {
			j.logger.Printf("news sync processed=%d fetched=%d inserted=%d failures=%d",
				result.SourcesProcessed, result.ArticlesFetched, result.ArticlesInserted, result.Failures)
		}
	}
	return result, err
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
