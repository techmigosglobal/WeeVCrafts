package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

// SearchMetrics is an injected, bounded in-process observer. It exposes
// operational counters without making search correctness depend on metrics.
type SearchMetrics struct {
	cacheHits       atomic.Int64
	postgresQueries atomic.Int64
	meiliQueries    atomic.Int64
	errors          atomic.Int64
	observations    atomic.Int64
	resultCount     atomic.Int64
	durationNanos   atomic.Int64
}

type SearchSnapshot struct {
	Observations    int64 `json:"observations"`
	CacheHits       int64 `json:"cache_hits"`
	PostgresQueries int64 `json:"postgres_queries"`
	MeiliQueries    int64 `json:"meilisearch_queries"`
	Errors          int64 `json:"errors"`
	ResultCount     int64 `json:"result_count"`
	DurationMillis  int64 `json:"duration_millis"`
}

func NewSearchMetrics() *SearchMetrics { return &SearchMetrics{} }

func (m *SearchMetrics) ObserveSearch(_ context.Context, observation ports.SearchObservation) {
	m.observations.Add(1)
	m.resultCount.Add(int64(observation.ResultCount))
	m.durationNanos.Add(observation.Duration.Nanoseconds())
	if observation.CacheHit {
		m.cacheHits.Add(1)
	}
	if observation.Error {
		m.errors.Add(1)
	}
	switch observation.Backend {
	case "postgresql":
		m.postgresQueries.Add(1)
	case "index":
		m.meiliQueries.Add(1)
	}
}

func (m *SearchMetrics) Snapshot() SearchSnapshot {
	return SearchSnapshot{
		Observations:    m.observations.Load(),
		CacheHits:       m.cacheHits.Load(),
		PostgresQueries: m.postgresQueries.Load(),
		MeiliQueries:    m.meiliQueries.Load(),
		Errors:          m.errors.Load(),
		ResultCount:     m.resultCount.Load(),
		DurationMillis:  time.Duration(m.durationNanos.Load()).Milliseconds(),
	}
}
