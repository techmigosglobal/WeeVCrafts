package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeRateLimiter struct {
	allowed bool
	err     error
	keys    []string
}

func (f *fakeRateLimiter) Allow(_ context.Context, key string, _ int, _ time.Duration) (bool, error) {
	f.keys = append(f.keys, key)
	return f.allowed, f.err
}

func TestAuthRateLimitProtectsOnlyAuthenticationRoutes(t *testing.T) {
	limiter := &fakeRateLimiter{allowed: false}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := authRateLimit(next, limiter)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	request.RemoteAddr = "192.0.2.10:4000"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected auth request to be rate limited, got %d", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") != "60" || !strings.Contains(recorder.Body.String(), `"RATE_LIMITED"`) {
		t.Fatalf("expected retry/error contract, headers=%v body=%s", recorder.Header(), recorder.Body.String())
	}
	if len(limiter.keys) != 1 || !strings.HasPrefix(limiter.keys[0], "auth:192.0.2.10") {
		t.Fatalf("unexpected limiter key: %v", limiter.keys)
	}

	limiter.allowed = true
	request = httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	request.RemoteAddr = "192.0.2.10:4000"
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("catalog request should bypass auth limiter, got %d", recorder.Code)
	}
	if len(limiter.keys) != 1 {
		t.Fatalf("catalog request unexpectedly consumed auth limit: %v", limiter.keys)
	}
}

func TestAuthRateLimitUnavailableReturnsJSONServiceError(t *testing.T) {
	limiter := &fakeRateLimiter{err: context.DeadlineExceeded}
	handler := authRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), limiter)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{}`))
	request.RemoteAddr = "192.0.2.11:4000"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected limiter outage status 503, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"RATE_LIMIT_UNAVAILABLE"`) {
		t.Fatalf("expected structured outage error, got %s", recorder.Body.String())
	}
}
