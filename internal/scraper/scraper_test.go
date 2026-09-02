package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/voton4ik/miniprom/internal/config"
	"github.com/voton4ik/miniprom/internal/storage"
)

func newManager(t *testing.T, url string) *Manager {
	t.Helper()
	cfg := &config.Config{
		Targets: []config.Target{{Job: "test", URL: url}},
	}
	cfg.ScrapeInterval.Duration = time.Second
	cfg.Timeout.Duration = time.Second
	return New(storage.New(time.Hour), cfg)
}

func TestScrapeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("http_requests_total{code=\"200\"} 42\nroom_temp 21.5\n"))
	}))
	defer srv.Close()

	m := newManager(t, srv.URL)
	target := m.targets[0]
	m.scrape(context.Background(), target)

	series := m.store.Latest("http_requests_total", nil)
	if len(series) != 1 || series[0].Samples[0].V != 42 {
		t.Fatalf("метрика не сохранена корректно: %+v", series)
	}
	if series[0].Labels["job"] != "test" || series[0].Labels["instance"] != srv.URL {
		t.Errorf("метки job/instance не проставлены: %v", series[0].Labels)
	}

	state := m.Targets()[0]
	if !state.Up || state.Samples != 2 {
		t.Errorf("состояние цели неверно: %+v", state)
	}
	if up := m.store.Latest("up", nil); len(up) != 1 || up[0].Samples[0].V != 1 {
		t.Errorf("метрика up должна быть 1")
	}
}

func TestScrapeServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := newManager(t, srv.URL)
	m.scrape(context.Background(), m.targets[0])

	state := m.Targets()[0]
	if state.Up {
		t.Error("цель не должна считаться живой при 500")
	}
	if state.LastError == "" {
		t.Error("ожидалась зафиксированная ошибка")
	}
	if up := m.store.Latest("up", nil); up[0].Samples[0].V != 0 {
		t.Error("метрика up должна быть 0")
	}
}

func TestScrapeTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("x 1\n"))
	}))
	defer srv.Close()

	m := newManager(t, srv.URL)
	m.client.Timeout = 20 * time.Millisecond
	m.scrape(context.Background(), m.targets[0])

	if m.Targets()[0].Up {
		t.Error("при таймауте цель должна быть недоступна")
	}
}

func TestReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("x 1\n"))
	}))
	defer srv.Close()

	m := newManager(t, srv.URL)
	if m.Ready() {
		t.Error("до первого сбора сервис не готов")
	}
	m.scrape(context.Background(), m.targets[0])
	if !m.Ready() {
		t.Error("после сбора сервис должен быть готов")
	}
}
