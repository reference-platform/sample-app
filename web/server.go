package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed index.html
var indexHTML string

var page = template.Must(template.New("index").Parse(indexHTML))

type note struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type server struct {
	api    string
	client *http.Client
	log    *slog.Logger
}

func newServer(api string, log *slog.Logger) http.Handler {
	s := &server{
		api:    strings.TrimRight(api, "/"),
		client: &http.Client{Timeout: 3 * time.Second},
		log:    log,
	}
	mux := http.NewServeMux()
	// {$} matches only "/", not every path.
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("POST /notes", s.addNote)
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	return s.logRequests(mux)
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Notes []note
		Error string
	}{}
	status := http.StatusOK

	notes, err := s.listNotes(r.Context())
	if err != nil {
		s.log.Error("list notes failed", "error", err)
		data.Error = "Notes are not available right now."
		status = http.StatusBadGateway
	}
	data.Notes = notes

	// Render before writing, so a template error can still return a clean 500.
	var buf bytes.Buffer
	if err := page.Execute(&buf, data); err != nil {
		s.log.Error("render failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (s *server) addNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	text := strings.TrimSpace(r.FormValue("text"))
	if n := utf8.RuneCountInString(text); n < 1 || n > 500 {
		http.Error(w, "text must be 1 to 500 characters", http.StatusBadRequest)
		return
	}

	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := s.call(r.Context(), http.MethodPost, "/api/notes", bytes.NewReader(body), http.StatusCreated)
	if err != nil {
		s.log.Error("add note failed", "error", err)
		http.Error(w, "could not save the note", http.StatusBadGateway)
		return
	}
	resp.Body.Close()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) listNotes(ctx context.Context) ([]note, error) {
	resp, err := s.call(ctx, http.MethodGet, "/api/notes?limit=20", nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var notes []note
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		return nil, err
	}
	return notes, nil
}

// call sends a request to the api. It returns the response only when the
// status is the wanted one. The caller closes the body.
func (s *server) call(ctx context.Context, method, path string, body io.Reader, want int) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.api+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != want {
		resp.Body.Close()
		return nil, fmt.Errorf("api returned %s", resp.Status)
	}
	return resp, nil
}

func (s *server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")
	resp, err := s.call(ctx, http.MethodGet, "/readyz", nil, http.StatusOK)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"api not ready"}`))
		return
	}
	resp.Body.Close()
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
