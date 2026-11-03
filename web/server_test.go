package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type fakeAPI struct {
	*httptest.Server
	posted []string
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/notes", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"text":"<b>hello</b>","created_at":"2026-10-29T10:00:00Z"}]`))
	})
	mux.HandleFunc("POST /api/notes", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Text string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.posted = append(f.posted, in.Text)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func newTestServer(t *testing.T, api string) *httptest.Server {
	t.Helper()
	h := newServer(api, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func TestIndexShowsEscapedNotes(t *testing.T) {
	srv := newTestServer(t, newFakeAPI(t).URL)
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "&lt;b&gt;hello&lt;/b&gt;") {
		t.Fatalf("note not rendered and escaped: %s", body)
	}
}

func TestIndexWhenAPIIsDown(t *testing.T) {
	api := newFakeAPI(t)
	srv := newTestServer(t, api.URL)
	api.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "not available") {
		t.Fatalf("missing error text: %s", body)
	}
}

func TestAddNoteForwardsToAPIAndRedirects(t *testing.T) {
	api := newFakeAPI(t)
	srv := newTestServer(t, api.URL)
	resp, err := noRedirect().PostForm(srv.URL+"/notes", url.Values{"text": {" hi "}})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Fatalf("status = %d, location = %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if len(api.posted) != 1 || api.posted[0] != "hi" {
		t.Fatalf("api received %v", api.posted)
	}
}

func TestAddNoteRejectsEmptyText(t *testing.T) {
	api := newFakeAPI(t)
	srv := newTestServer(t, api.URL)
	resp, err := http.PostForm(srv.URL+"/notes", url.Values{"text": {"  "}})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || len(api.posted) != 0 {
		t.Fatalf("status = %d, api received %v", resp.StatusCode, api.posted)
	}
}

func TestProbes(t *testing.T) {
	api := newFakeAPI(t)
	srv := newTestServer(t, api.URL)

	for path, want := range map[string]int{"/healthz": 200, "/readyz": 200} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("%s = %d, want %d", path, resp.StatusCode, want)
		}
	}

	api.Close()
	for path, want := range map[string]int{"/healthz": 200, "/readyz": 503} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("api down: %s = %d, want %d", path, resp.StatusCode, want)
		}
	}
}
