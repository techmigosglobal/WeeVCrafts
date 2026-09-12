package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

type indexEvents struct {
	events      []ports.OutboxEvent
	claimedType string
	acked       []int64
	failed      []int64
}

func (e *indexEvents) Publish(context.Context, ports.OutboxEventInput) (ports.OutboxEvent, error) {
	return ports.OutboxEvent{}, nil
}
func (e *indexEvents) Claim(context.Context, int) ([]ports.OutboxEvent, error) { return nil, nil }
func (e *indexEvents) ClaimByType(_ context.Context, eventType string, _ int) ([]ports.OutboxEvent, error) {
	e.claimedType = eventType
	return e.events, nil
}
func (e *indexEvents) Ack(_ context.Context, id int64) error {
	e.acked = append(e.acked, id)
	return nil
}
func (e *indexEvents) Fail(_ context.Context, id int64, _ time.Duration, _ string) error {
	e.failed = append(e.failed, id)
	return nil
}

type indexSource struct {
	documents []ports.SearchDocument
	err       error
}

func (s *indexSource) ListApprovedForIndex(context.Context) ([]ports.SearchDocument, error) {
	return s.documents, s.err
}

type indexer struct {
	calls     int
	documents []ports.SearchDocument
	err       error
}

type cacheInvalidator struct {
	prefixes []string
	err      error
}

func (c *cacheInvalidator) DeletePrefix(_ context.Context, prefix string) error {
	c.prefixes = append(c.prefixes, prefix)
	return c.err
}

func (i *indexer) Rebuild(_ context.Context, documents []ports.SearchDocument) error {
	i.calls++
	i.documents = documents
	return i.err
}

func TestProductIndexProcessorRebuildsAndAcknowledgesProductEvents(t *testing.T) {
	events := &indexEvents{events: []ports.OutboxEvent{{ID: 9, EventType: ProductIndexEventType}}}
	source := &indexSource{documents: []ports.SearchDocument{{Slug: "soap"}}}
	searchIndexer := &indexer{}
	processor := NewProductIndexProcessor(events, source, searchIndexer)

	processed, err := processor.Process(context.Background(), 10)
	if err != nil || processed != 1 {
		t.Fatalf("process product index event: processed=%d err=%v", processed, err)
	}
	if events.claimedType != ProductIndexEventType || len(events.acked) != 1 || events.acked[0] != 9 || searchIndexer.calls != 1 || len(searchIndexer.documents) != 1 {
		t.Fatalf("unexpected index processing state: events=%+v indexer=%+v", events, searchIndexer)
	}
}

func TestProductIndexProcessorInvalidatesSearchCacheBeforeAcknowledging(t *testing.T) {
	events := &indexEvents{events: []ports.OutboxEvent{{ID: 11, EventType: ProductIndexEventType}}}
	invalidator := &cacheInvalidator{}
	processed, err := NewProductIndexProcessorWithCache(events, &indexSource{}, &indexer{}, invalidator).Process(context.Background(), 1)
	if err != nil || processed != 1 || len(invalidator.prefixes) != 1 || invalidator.prefixes[0] != "catalog:search:" || len(events.acked) != 1 {
		t.Fatalf("expected cache invalidation before ack: processed=%d err=%v invalidator=%+v events=%+v", processed, err, invalidator, events)
	}
}

func TestProductIndexProcessorRequeuesFailedRebuild(t *testing.T) {
	events := &indexEvents{events: []ports.OutboxEvent{{ID: 10, EventType: ProductIndexEventType}}}
	source := &indexSource{documents: []ports.SearchDocument{{Slug: "soap"}}}
	searchIndexer := &indexer{err: errors.New("search index unavailable")}
	processed, err := NewProductIndexProcessor(events, source, searchIndexer).Process(context.Background(), 1)
	if err == nil || processed != 0 || len(events.failed) != 1 || len(events.acked) != 0 {
		t.Fatalf("expected failed event to be requeued: processed=%d err=%v events=%+v", processed, err, events)
	}
}

func TestProductIndexProcessorIgnoresStaleEventPayloadAndUsesAuthoritativeSource(t *testing.T) {
	events := &indexEvents{events: []ports.OutboxEvent{{ID: 12, EventType: ProductIndexEventType, Payload: []byte(`{"status":"approved","name":"old"}`)}}}
	source := &indexSource{documents: []ports.SearchDocument{}}
	searchIndexer := &indexer{}
	processed, err := NewProductIndexProcessor(events, source, searchIndexer).Process(context.Background(), 1)
	if err != nil || processed != 1 || len(searchIndexer.documents) != 0 || len(events.acked) != 1 {
		t.Fatalf("stale event was allowed to publish stale data: processed=%d docs=%+v events=%+v err=%v", processed, searchIndexer.documents, events, err)
	}
}
