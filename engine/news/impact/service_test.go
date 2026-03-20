package impact

import (
	"context"
	"testing"
	"time"
)

type stubImpactStore struct {
	candidates      []Candidate
	rows            []ImpactRow
	upserts         int
	snapshotUpdates []string
	histories       int
}

func (s *stubImpactStore) ListCandidates(ctx context.Context, limit int) ([]Candidate, error) {
	return append([]Candidate(nil), s.candidates...), nil
}

func (s *stubImpactStore) UpsertCandidate(ctx context.Context, candidate Candidate, priceAtPublish *float64) error {
	s.upserts++
	return nil
}

func (s *stubImpactStore) ListPendingRows(ctx context.Context, limit int) ([]ImpactRow, error) {
	return append([]ImpactRow(nil), s.rows...), nil
}

func (s *stubImpactStore) UpdateSnapshot(ctx context.Context, articleID int64, coinID, window string, price float64) error {
	s.snapshotUpdates = append(s.snapshotUpdates, window)
	return nil
}

func (s *stubImpactStore) InsertHistory(ctx context.Context, row ImpactRow) error {
	s.histories++
	return nil
}

type stubPriceResolver struct {
	base time.Time
}

func (s stubPriceResolver) Resolve(ctx context.Context, coinID string, at time.Time) (PriceSample, error) {
	switch {
	case at.Equal(s.base):
		return PriceSample{CoinID: coinID, Exchange: "binance", Symbol: "BTC/USDT", Price: 100, Timestamp: at}, nil
	case at.Equal(s.base.Add(time.Hour)):
		return PriceSample{CoinID: coinID, Exchange: "binance", Symbol: "BTC/USDT", Price: 110, Timestamp: at}, nil
	case at.Equal(s.base.Add(4 * time.Hour)):
		return PriceSample{CoinID: coinID, Exchange: "binance", Symbol: "BTC/USDT", Price: 120, Timestamp: at}, nil
	case at.Equal(s.base.Add(24 * time.Hour)):
		return PriceSample{CoinID: coinID, Exchange: "binance", Symbol: "BTC/USDT", Price: 130, Timestamp: at}, nil
	default:
		return PriceSample{}, ErrPriceNotFound
	}
}

func TestServiceRunOnce(t *testing.T) {
	publishedAt := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	store := &stubImpactStore{
		candidates: []Candidate{{
			ArticleID:   1,
			CoinID:      "bitcoin",
			PublishedAt: publishedAt,
			Sentiment:   "bullish",
			Category:    "defi",
		}},
		rows: []ImpactRow{{
			ArticleID:       1,
			CoinID:          "bitcoin",
			PublishedAt:     publishedAt,
			Snapshot1hDone:  false,
			Snapshot4hDone:  false,
			Snapshot24hDone: false,
			Sentiment:       "bullish",
			Category:        "defi",
			IsBreaking:      true,
		}},
	}
	svc := NewService(store, stubPriceResolver{base: publishedAt}, 10)
	svc.now = func() time.Time { return publishedAt.Add(25 * time.Hour) }

	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.CandidatesUpserted != 1 {
		t.Fatalf("CandidatesUpserted = %d, want 1", result.CandidatesUpserted)
	}
	if result.SnapshotsUpdated != 3 {
		t.Fatalf("SnapshotsUpdated = %d, want 3", result.SnapshotsUpdated)
	}
	if result.HistoryInserted != 1 {
		t.Fatalf("HistoryInserted = %d, want 1", result.HistoryInserted)
	}
	if store.upserts != 2 {
		t.Fatalf("upserts = %d, want 2", store.upserts)
	}
	if len(store.snapshotUpdates) != 3 {
		t.Fatalf("snapshot updates = %d, want 3", len(store.snapshotUpdates))
	}
	if store.histories != 1 {
		t.Fatalf("histories = %d, want 1", store.histories)
	}
}
