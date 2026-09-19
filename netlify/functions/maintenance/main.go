// Package main runs one short, bounded application-maintenance batch when
// invoked by Netlify's scheduled-functions service.
package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/wecratfs/commerce/internal/apphost"
)

var runtimeMu sync.Mutex
var applicationRuntime *apphost.Runtime

func getRuntime() (*apphost.Runtime, error) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if applicationRuntime != nil {
		return applicationRuntime, nil
	}
	initialized, err := apphost.NewNetlify(slog.Default())
	if err != nil {
		return nil, err
	}
	applicationRuntime = initialized
	return applicationRuntime, nil
}

func handler(ctx context.Context, _ events.CloudWatchEvent) error {
	runtime, err := getRuntime()
	if err != nil {
		slog.Error("initialize scheduled Netlify runtime", "error", err)
		return err
	}
	jobContext, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	return runtime.RunMaintenance(jobContext)
}

func main() {
	lambda.Start(handler)
}
