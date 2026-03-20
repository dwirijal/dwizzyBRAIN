package archive

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
		interval = 6 * time.Hour
	}
	return &Job{service: service, interval: interval, logger: logger}
}

func (j *Job) RunOnce(ctx context.Context) (Result, error) {
	result, err := j.service.RunOnce(ctx)
	if j.logger != nil {
		if len(result.FailedArticles) > 0 {
			j.logger.Printf("news markdown export scanned=%d exported=%d failures=%d failed_articles=%v",
				result.ArticlesScanned, result.ArticlesExported, result.Failures, result.FailedArticles)
		} else {
			j.logger.Printf("news markdown export scanned=%d exported=%d failures=%d",
				result.ArticlesScanned, result.ArticlesExported, result.Failures)
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
