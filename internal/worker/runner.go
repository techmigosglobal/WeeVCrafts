package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrInvalidConfiguration = errors.New("worker configuration must use positive concurrency and queue values")
	ErrNotStarted           = errors.New("worker has not been started")
	ErrStopped              = errors.New("worker has been stopped")
)

type Job func(context.Context) error

// Runner executes a finite number of jobs and applies backpressure through a
// bounded queue. It is intentionally generic; business workflows decide which
// jobs are submitted in later phases.
type Runner struct {
	jobs        chan Job
	stop        chan struct{}
	done        chan struct{}
	concurrency int
	started     atomic.Bool
	startOnce   sync.Once
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func New(concurrency, queue int) (*Runner, error) {
	if concurrency < 1 || queue < 1 {
		return nil, ErrInvalidConfiguration
	}
	return &Runner{
		jobs:        make(chan Job, queue),
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		concurrency: concurrency,
	}, nil
}

func (r *Runner) Start(ctx context.Context) {
	r.startOnce.Do(func() {
		r.started.Store(true)
		r.wg.Add(r.concurrency)
		for i := 0; i < r.concurrency; i++ {
			go r.run(ctx)
		}
		go func() {
			select {
			case <-ctx.Done():
				r.stopOnce.Do(func() { close(r.stop) })
			case <-r.done:
			}
		}()
		go func() {
			r.wg.Wait()
			close(r.done)
		}()
	})
}

func (r *Runner) Submit(ctx context.Context, job Job) error {
	if !r.started.Load() {
		return ErrNotStarted
	}
	if job == nil {
		return errors.New("worker job cannot be nil")
	}
	select {
	case <-r.stop:
		return ErrStopped
	default:
	}
	select {
	case <-r.stop:
		return ErrStopped
	case <-ctx.Done():
		return ctx.Err()
	case r.jobs <- job:
		return nil
	}
}

func (r *Runner) Shutdown(ctx context.Context) error {
	if !r.started.Load() {
		return ErrNotStarted
	}
	r.stopOnce.Do(func() { close(r.stop) })
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) run(ctx context.Context) {
	defer r.wg.Done()
	for {
		select {
		case <-r.stop:
			return
		case job := <-r.jobs:
			if ctx.Err() != nil {
				return
			}
			_ = job(ctx)
		}
	}
}
