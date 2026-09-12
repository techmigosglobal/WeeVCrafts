package catalog

import (
	"context"
	"errors"
	"testing"

	domain "github.com/wecratfs/commerce/internal/domain/catalog"
	"github.com/wecratfs/commerce/internal/ports"
)

type fakeRepository struct {
	products []domain.Product
	listErr  error
	getErr   error
	seenCtx  context.Context
}

func (f *fakeRepository) ListApproved(ctx context.Context, _ string, _ int) ([]domain.Product, error) {
	f.seenCtx = ctx
	return f.products, f.listErr
}

func (f *fakeRepository) GetApprovedBySlug(ctx context.Context, _ string) (domain.Product, error) {
	f.seenCtx = ctx
	if f.getErr != nil {
		return domain.Product{}, f.getErr
	}
	return f.products[0], nil
}

func TestListRejectsInvalidLimit(t *testing.T) {
	service := NewService(&fakeRepository{})

	if _, err := service.List(context.Background(), "", 0); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("expected invalid limit error, got %v", err)
	}
	if _, err := service.List(context.Background(), "", 101); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("expected invalid limit error, got %v", err)
	}
}

func TestListPropagatesContextAndRepositoryResult(t *testing.T) {
	repository := &fakeRepository{products: []domain.Product{{Slug: "real-product"}}}
	service := NewService(repository)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	products, err := service.List(ctx, "natural-care", 24)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	if len(products) != 1 || products[0].Slug != "real-product" {
		t.Fatalf("unexpected products: %#v", products)
	}
	if repository.seenCtx != ctx {
		t.Fatal("application service did not propagate context")
	}
}

func TestGetPropagatesNotFound(t *testing.T) {
	service := NewService(&fakeRepository{getErr: ports.ErrNotFound})

	if _, err := service.Get(context.Background(), "missing"); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
