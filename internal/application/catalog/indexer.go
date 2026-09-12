package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

const ProductIndexEventType = "ProductIndexRequested"

var ErrIndexWorkerUnavailable = errors.New("product index worker is unavailable")

type ProductIndexProcessor struct {
	events      ports.OutboxStore
	source      ports.ApprovedIndexSource
	indexer     ports.SearchIndexer
	invalidator ports.CacheInvalidator
}

func NewProductIndexProcessor(events ports.OutboxStore, source ports.ApprovedIndexSource, indexer ports.SearchIndexer) *ProductIndexProcessor {
	return NewProductIndexProcessorWithCache(events, source, indexer, nil)
}

func NewProductIndexProcessorWithCache(events ports.OutboxStore, source ports.ApprovedIndexSource, indexer ports.SearchIndexer, invalidator ports.CacheInvalidator) *ProductIndexProcessor {
	return &ProductIndexProcessor{events: events, source: source, indexer: indexer, invalidator: invalidator}
}

func (p *ProductIndexProcessor) Process(ctx context.Context, limit int) (processed int, err error) {
	if p.events == nil || p.source == nil || p.indexer == nil {
		return 0, ErrIndexWorkerUnavailable
	}
	events, err := p.events.ClaimByType(ctx, ProductIndexEventType, limit)
	if err != nil {
		return 0, err
	}
	for _, event := range events {
		documents, sourceErr := p.source.ListApprovedForIndex(ctx)
		if sourceErr != nil {
			_ = p.events.Fail(ctx, event.ID, time.Minute, sourceErr.Error())
			if err == nil {
				err = sourceErr
			}
			continue
		}
		if indexErr := p.indexer.Rebuild(ctx, documents); indexErr != nil {
			_ = p.events.Fail(ctx, event.ID, time.Minute, indexErr.Error())
			if err == nil {
				err = indexErr
			}
			continue
		}
		if p.invalidator != nil {
			if invalidateErr := p.invalidator.DeletePrefix(ctx, "catalog:search:"); invalidateErr != nil {
				_ = p.events.Fail(ctx, event.ID, time.Minute, invalidateErr.Error())
				if err == nil {
					err = invalidateErr
				}
				continue
			}
		}
		if ackErr := p.events.Ack(ctx, event.ID); ackErr != nil {
			if err == nil {
				err = fmt.Errorf("ack product index event %d: %w", event.ID, ackErr)
			}
			continue
		}
		processed++
	}
	return processed, err
}
