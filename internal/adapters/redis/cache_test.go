package redis

import (
	"context"
	"testing"
)

func TestCacheImplementsBoundedSearchInvalidationPort(t *testing.T) {
	var _ interface {
		DeletePrefix(context.Context, string) error
	} = (*Cache)(nil)
}
