package worker

import (
	"context"
	"time"
)

// RunPeriodic owns one bounded scheduler goroutine. Each invocation is still
// responsible for applying its own context timeout and work limit.
func RunPeriodic(ctx context.Context, interval time.Duration, job Job) {
	if interval <= 0 || job == nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = job(ctx)
		}
	}
}
