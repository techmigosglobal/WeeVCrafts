package catalog

import (
	"context"
	"errors"
	"strings"

	domain "github.com/wecratfs/commerce/internal/domain/catalog"
	"github.com/wecratfs/commerce/internal/ports"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidLimit    = errors.New("product limit must be between 1 and 100")
)

type Service struct {
	repository ports.ProductRepository
}

func NewService(repository ports.ProductRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, categorySlug string, limit int) ([]domain.Product, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidLimit
	}
	return s.repository.ListApproved(ctx, strings.TrimSpace(categorySlug), limit)
}

func (s *Service) Get(ctx context.Context, slug string) (domain.Product, error) {
	product, err := s.repository.GetApprovedBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return domain.Product{}, ErrProductNotFound
		}
		return domain.Product{}, err
	}
	return product, nil
}
