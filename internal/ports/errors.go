package ports

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrCacheMiss = errors.New("cache miss")
)
