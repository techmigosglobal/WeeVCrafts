package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerRejectsInvalidConfiguration(t *testing.T) {
	if _, err := New(0, 1); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("expected invalid concurrency error, got %v", err)
	}
	if _, err := New(1, 0); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("expected invalid queue error, got %v", err)
	}
}

func TestRunnerAppliesBoundedConcurrencyAndShutsDown(t *testing.T) {
	runner, err := New(2, 2)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	if err := runner.Submit(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("expected not-started error, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner.Start(ctx)

	var running atomic.Int32
	var maximum atomic.Int32
	job := func(context.Context) error {
		current := running.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		running.Add(-1)
		return nil
	}
	for i := 0; i < 4; i++ {
		if err := runner.Submit(context.Background(), job); err != nil {
			t.Fatalf("submit job %d: %v", i, err)
		}
	}

	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := runner.Shutdown(shutdownContext); err != nil {
		t.Fatalf("shutdown runner: %v", err)
	}
	if maximum.Load() > 2 {
		t.Fatalf("runner exceeded configured concurrency: %d", maximum.Load())
	}
	if err := runner.Submit(context.Background(), job); !errors.Is(err, ErrStopped) {
		t.Fatalf("expected stopped error, got %v", err)
	}
}

func TestRunnerPropagatesCancellationToJob(t *testing.T) {
	runner, err := New(1, 1)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	runner.Start(ctx)

	started := make(chan struct{})
	finished := make(chan struct{})
	if err := runner.Submit(context.Background(), func(jobContext context.Context) error {
		close(started)
		<-jobContext.Done()
		close(finished)
		return nil
	}); err != nil {
		t.Fatalf("submit job: %v", err)
	}
	<-started
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("job did not receive cancellation")
	}

	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := runner.Shutdown(shutdownContext); err != nil {
		t.Fatalf("shutdown runner: %v", err)
	}
}

func TestRunPeriodicStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunPeriodic(ctx, time.Millisecond, func(context.Context) error {
			cancel()
			return nil
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("periodic runner did not stop after cancellation")
	}
}
