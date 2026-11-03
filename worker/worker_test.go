package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const interval = 10 * time.Second

type harness struct {
	w   *worker
	api *httptest.Server
	log *bytes.Buffer
	t0  time.Time
	cur time.Time
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	api := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/notes" {
			http.NotFound(rw, r)
			return
		}
		_, _ = rw.Write([]byte(`[{"text":"a"},{"text":"b"}]`))
	}))
	t.Cleanup(api.Close)

	h := &harness{api: api, log: &bytes.Buffer{}, t0: time.Unix(1_000_000, 0)}
	h.cur = h.t0
	h.w = newWorker(api.URL, interval, slog.New(slog.NewJSONHandler(h.log, nil)))
	h.w.now = func() time.Time { return h.cur }
	return h
}

func (h *harness) status(path string) int {
	rec := httptest.NewRecorder()
	h.w.probes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

func TestTickCountsNotes(t *testing.T) {
	h := newHarness(t)
	h.w.tick(context.Background())
	if !strings.Contains(h.log.String(), `"count":2`) {
		t.Fatalf("log = %s", h.log)
	}
}

func TestProbesBeforeTheFirstTick(t *testing.T) {
	h := newHarness(t)
	if h.status("/healthz") != 503 || h.status("/readyz") != 503 {
		t.Fatal("both probes must fail before the first tick")
	}
}

func TestProbesAfterASuccessfulTick(t *testing.T) {
	h := newHarness(t)
	h.w.tick(context.Background())
	if h.status("/healthz") != 200 || h.status("/readyz") != 200 {
		t.Fatal("both probes must pass after a good tick")
	}
}

func TestProbesGoStaleWhenTheLoopStops(t *testing.T) {
	h := newHarness(t)
	h.w.tick(context.Background())
	h.cur = h.t0.Add(4 * interval)
	if h.status("/healthz") != 503 || h.status("/readyz") != 503 {
		t.Fatal("both probes must fail when no tick ran for three intervals")
	}
}

func TestAPIDownKeepsLivenessButFailsReadiness(t *testing.T) {
	h := newHarness(t)
	h.api.Close()
	h.w.tick(context.Background())
	if h.status("/healthz") != 200 {
		t.Fatal("healthz must pass while the loop runs")
	}
	if h.status("/readyz") != 503 {
		t.Fatal("readyz must fail when the api is down")
	}
}
