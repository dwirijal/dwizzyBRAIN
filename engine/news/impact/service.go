package impact

import (
	"context"
	"fmt"
	"time"
)

type store interface {
	ListCandidates(context.Context, int) ([]Candidate, error)
	UpsertCandidate(context.Context, Candidate, *float64) error
	ListPendingRows(context.Context, int) ([]ImpactRow, error)
	UpdateSnapshot(context.Context, int64, string, string, float64) error
	InsertHistory(context.Context, ImpactRow) error
}

type priceResolver interface {
	Resolve(context.Context, string, time.Time) (PriceSample, error)
}

type Service struct {
	store  store
	pricer priceResolver
	limit  int
	now    func() time.Time
}

func NewService(store store, pricer priceResolver, limit int) *Service {
	if limit <= 0 {
		limit = 50
	}
	return &Service{
		store:  store,
		pricer: pricer,
		limit:  limit,
		now:    time.Now,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.store == nil {
		return Result{}, fmt.Errorf("impact store is required")
	}
	if s.pricer == nil {
		return Result{}, fmt.Errorf("price resolver is required")
	}

	result := Result{}
	candidates, err := s.store.ListCandidates(ctx, s.limit)
	if err != nil {
		return result, err
	}
	for _, candidate := range candidates {
		var price *float64
		sample, err := s.pricer.Resolve(ctx, candidate.CoinID, candidate.PublishedAt)
		if err == nil {
			value := sample.Price
			price = &value
		}
		if err := s.store.UpsertCandidate(ctx, candidate, price); err != nil {
			result.Failures++
			result.FailedArticles = append(result.FailedArticles, candidate.ArticleID)
			continue
		}
		result.CandidatesUpserted++
	}

	rows, err := s.store.ListPendingRows(ctx, s.limit)
	if err != nil {
		return result, err
	}

	now := s.now().UTC()
	for _, row := range rows {
		if row.PriceAtPublish == nil {
			if sample, err := s.pricer.Resolve(ctx, row.CoinID, row.PublishedAt); err == nil {
				if err := s.store.UpsertCandidate(ctx, Candidate{
					ArticleID:   row.ArticleID,
					CoinID:      row.CoinID,
					PublishedAt: row.PublishedAt,
					Sentiment:   row.Sentiment,
					Importance:  row.Importance,
					Category:    row.Category,
					IsBreaking:  row.IsBreaking,
				}, &sample.Price); err == nil {
					row.PriceAtPublish = &sample.Price
				}
			}
		}

		if row.PriceAtPublish == nil {
			continue
		}

		if !row.Snapshot1hDone && now.After(row.PublishedAt.Add(time.Hour)) {
			if sample, err := s.pricer.Resolve(ctx, row.CoinID, row.PublishedAt.Add(time.Hour)); err == nil {
				if err := s.store.UpdateSnapshot(ctx, row.ArticleID, row.CoinID, "1h", sample.Price); err == nil {
					row.Price1h = &sample.Price
					row.Snapshot1hDone = true
					result.SnapshotsUpdated++
				}
			}
		}
		if !row.Snapshot4hDone && now.After(row.PublishedAt.Add(4*time.Hour)) {
			if sample, err := s.pricer.Resolve(ctx, row.CoinID, row.PublishedAt.Add(4*time.Hour)); err == nil {
				if err := s.store.UpdateSnapshot(ctx, row.ArticleID, row.CoinID, "4h", sample.Price); err == nil {
					row.Price4h = &sample.Price
					row.Snapshot4hDone = true
					result.SnapshotsUpdated++
				}
			}
		}
		if !row.Snapshot24hDone && now.After(row.PublishedAt.Add(24*time.Hour)) {
			if sample, err := s.pricer.Resolve(ctx, row.CoinID, row.PublishedAt.Add(24*time.Hour)); err == nil {
				if err := s.store.UpdateSnapshot(ctx, row.ArticleID, row.CoinID, "24h", sample.Price); err == nil {
					row.Price24h = &sample.Price
					row.Snapshot24hDone = true
					result.SnapshotsUpdated++
				}
			}
		}

		if row.Snapshot1hDone && row.Snapshot4hDone && row.Snapshot24hDone {
			if err := s.store.InsertHistory(ctx, row); err == nil {
				result.HistoryInserted++
			}
		}
	}

	return result, nil
}
