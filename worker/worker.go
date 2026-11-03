package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type worker struct {
	api      string
	interval time.Duration
	client   *http.Client
	log      *slog.Logger
	now      func() time.Time
	lastTick atomic.Int64
	lastOK   atomic.Int64
}

func newWorker(api string, interval time.Duration, log *slog.Logger) *worker {
	return &worker{
		api:      strings.TrimRight(api, "/"),
		interval: interval,
		client:   &http.Client{Timeout: 3 * time.Second},
		log:      log,
		now:      time.Now,
	}
}

func (w *worker) run(ctx context.Context) {
	w.tick(ctx)
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.tick(ctx)
		}
	}
}

func (w *worker) tick(ctx context.Context) {
	w.lastTick.Store(w.now().UnixNano())
	n, err := w.countRecentNotes(ctx)
	if err != nil {
		w.log.Error("count notes failed", "error", err)
		return
	}
	w.lastOK.Store(w.now().UnixNano())
	w.log.Info("recent notes counted", "count", n)
}

func (w *worker) countRecentNotes(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.api+"/api/notes?limit=200", nil)
	if err != nil {
		return 0, err
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("api returned %s", resp.Status)
	}
	var notes []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		return 0, err
	}
	return len(notes), nil
}

// fresh reports whether a timestamp is recent enough: within three intervals.
func (w *worker) fresh(at int64) bool {
	return at != 0 && w.now().Sub(time.Unix(0, at)) < 3*w.interval
}

func (w *worker) probes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(rw http.ResponseWriter, _ *http.Request) {
		reply(rw, w.fresh(w.lastTick.Load()), "loop stalled")
	})
	mux.HandleFunc("GET /readyz", func(rw http.ResponseWriter, _ *http.Request) {
		reply(rw, w.fresh(w.lastOK.Load()), "api not reachable")
	})
	return mux
}

func reply(rw http.ResponseWriter, ok bool, reason string) {
	rw.Header().Set("Content-Type", "application/json")
	if !ok {
		rw.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(rw, `{"status":%q}`, reason)
		return
	}
	_, _ = rw.Write([]byte(`{"status":"ok"}`))
}
