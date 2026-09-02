package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/voton4ik/miniprom/internal/config"
	"github.com/voton4ik/miniprom/internal/scraper"
	"github.com/voton4ik/miniprom/internal/storage"
)

func newTestServer(t *testing.T) (*httptest.Server, *storage.Storage) {
	t.Helper()
	store := storage.New(time.Hour)
	cfg := &config.Config{Targets: []config.Target{{Job: "test", URL: "http://example"}}}
	cfg.ScrapeInterval.Duration = time.Second
	cfg.Timeout.Duration = time.Second
	manager := scraper.New(store, cfg)
	srv := httptest.NewServer(New(store, manager).Handler())
	t.Cleanup(srv.Close)
	return srv, store
}

func TestQueryEndpoint(t *testing.T) {
	srv, store := newTestServer(t)
	store.Add("cpu", map[string]string{"host": "a"}, 5, time.Now())

	resp, err := http.Get(srv.URL + "/api/query?metric=cpu")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался 200, получен %d", resp.StatusCode)
	}
	var series []storage.Series
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		t.Fatal(err)
	}
	if len(series) != 1 || series[0].Samples[0].V != 5 {
		t.Errorf("неверный ответ: %+v", series)
	}
}

func TestQueryMissingMetric(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/query")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("ожидался 400, получен %d", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ожидался 200, получен %d", resp.StatusCode)
	}
}

func TestReadyEndpointNotReady(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("ожидался 503 до первого сбора, получен %d", resp.StatusCode)
	}
}

func TestUnknownPath(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ожидался 404, получен %d", resp.StatusCode)
	}
}
