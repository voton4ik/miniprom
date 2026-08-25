package scraper

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/voton4ik/miniprom/internal/config"
	"github.com/voton4ik/miniprom/internal/parser"
	"github.com/voton4ik/miniprom/internal/storage"
)

type TargetState struct {
	Job        string        `json:"job"`
	URL        string        `json:"url"`
	Up         bool          `json:"up"`
	LastScrape time.Time     `json:"last_scrape"`
	Duration   time.Duration `json:"duration_ms"`
	Samples    int           `json:"samples"`
	LastError  string        `json:"last_error"`
}

type Manager struct {
	store    *storage.Storage
	targets  []config.Target
	interval time.Duration
	client   *http.Client

	mu    sync.RWMutex
	state map[string]*TargetState
}

func New(store *storage.Storage, cfg *config.Config) *Manager {
	m := &Manager{
		store:    store,
		targets:  cfg.Targets,
		interval: cfg.ScrapeInterval.Duration,
		client:   &http.Client{Timeout: cfg.Timeout.Duration},
		state:    map[string]*TargetState{},
	}
	for _, t := range cfg.Targets {
		m.state[t.URL] = &TargetState{Job: t.Job, URL: t.URL}
	}
	return m
}

func (m *Manager) Start(ctx context.Context) {
	for _, t := range m.targets {
		go m.loop(ctx, t)
	}
}

func (m *Manager) loop(ctx context.Context, t config.Target) {
	m.scrape(ctx, t)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.scrape(ctx, t)
		}
	}
}

func (m *Manager) scrape(ctx context.Context, t config.Target) {
	start := time.Now()
	metrics, err := m.fetch(ctx, t.URL)
	elapsed := time.Since(start)
	ts := time.Now()

	st := &TargetState{
		Job:        t.Job,
		URL:        t.URL,
		LastScrape: ts,
		Duration:   elapsed,
	}

	up := 0.0
	if err == nil {
		up = 1.0
		st.Up = true
		st.Samples = len(metrics)
		for _, metric := range metrics {
			labels := enrich(metric.Labels, t)
			m.store.Add(metric.Name, labels, metric.Value, ts)
		}
	} else {
		st.LastError = err.Error()
	}

	m.store.Add("up", map[string]string{"job": t.Job, "instance": t.URL}, up, ts)
	m.store.Add("scrape_duration_seconds", map[string]string{"job": t.Job, "instance": t.URL}, elapsed.Seconds(), ts)

	m.mu.Lock()
	m.state[t.URL] = st
	m.mu.Unlock()
}

func (m *Manager) fetch(ctx context.Context, url string) ([]parser.Metric, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &httpError{status: resp.StatusCode}
	}
	return parser.Parse(resp.Body)
}

func (m *Manager) Targets() []TargetState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]TargetState, 0, len(m.state))
	for _, st := range m.state {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Job != out[j].Job {
			return out[i].Job < out[j].Job
		}
		return out[i].URL < out[j].URL
	})
	return out
}

func enrich(labels map[string]string, t config.Target) map[string]string {
	out := make(map[string]string, len(labels)+2)
	for k, v := range labels {
		out[k] = v
	}
	if _, ok := out["job"]; !ok {
		out["job"] = t.Job
	}
	if _, ok := out["instance"]; !ok {
		out["instance"] = t.URL
	}
	return out
}

type httpError struct {
	status int
}

func (e *httpError) Error() string {
	return "неожиданный код ответа: " + http.StatusText(e.status)
}
