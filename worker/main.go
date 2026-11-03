package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	api := os.Getenv("API_URL")
	if api == "" {
		log.Error("API_URL is required")
		os.Exit(1)
	}
	interval := 30 * time.Second
	if v := os.Getenv("INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			log.Error("INTERVAL must be a positive duration, for example 30s", "value", v)
			os.Exit(1)
		}
		interval = d
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	w := newWorker(api, interval, log)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           w.probes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go w.run(ctx)
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("started", "addr", srv.Addr, "interval", interval.String())

	select {
	case err := <-errc:
		log.Error("server failed", "error", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}
