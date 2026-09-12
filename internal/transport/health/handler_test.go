package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiveIsIndependentOfDependencies(t *testing.T) {
	handler := NewHandler(map[string]Probe{
		"database": func(context.Context) error { return errors.New("database is down") },
	})
	recorder := httptest.NewRecorder()
	handler.Live(recorder, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected live status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"live"`) {
		t.Fatalf("unexpected live body: %s", recorder.Body.String())
	}
}

func TestReadyReportsFailedProbe(t *testing.T) {
	handler := NewHandler(map[string]Probe{
		"database": func(context.Context) error { return errors.New("database is down") },
		"cache":    func(context.Context) error { return nil },
	})
	recorder := httptest.NewRecorder()
	handler.Ready(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected not-ready status 503, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"status":"not_ready"`) || !strings.Contains(body, `"database":"not_ready"`) {
		t.Fatalf("unexpected readiness body: %s", body)
	}
}
