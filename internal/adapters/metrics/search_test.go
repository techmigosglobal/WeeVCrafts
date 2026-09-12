package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestSearchMetricsSnapshotIsBoundedAndConcurrentSafe(t *testing.T) {
	metrics := NewSearchMetrics()
	metrics.ObserveSearch(context.Background(), ports.SearchObservation{Backend: "index", Duration: 4 * time.Millisecond, ResultCount: 2})
	metrics.ObserveSearch(context.Background(), ports.SearchObservation{Backend: "cache", CacheHit: true, Duration: time.Millisecond, ResultCount: 1})
	metrics.ObserveSearch(context.Background(), ports.SearchObservation{Backend: "postgresql", Error: true, Duration: 2 * time.Millisecond, ResultCount: 0})

	snapshot := metrics.Snapshot()
	if snapshot.Observations != 3 || snapshot.CacheHits != 1 || snapshot.MeiliQueries != 1 || snapshot.PostgresQueries != 1 || snapshot.Errors != 1 || snapshot.ResultCount != 3 || snapshot.DurationMillis != 7 {
		t.Fatalf("unexpected search metrics: %+v", snapshot)
	}
}
