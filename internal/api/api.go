package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/voton4ik/miniprom/internal/scraper"
	"github.com/voton4ik/miniprom/internal/storage"
)

type Server struct {
	store   *storage.Storage
	manager *scraper.Manager
	started time.Time
}

func New(store *storage.Storage, manager *scraper.Manager) *Server {
	return &Server{store: store, manager: manager, started: time.Now()}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/range", s.handleRange)
	mux.HandleFunc("/api/targets", s.handleTargets)
	mux.HandleFunc("/metrics", s.handleSelfMetrics)
	return logging(mux)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Names())
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("metric")
	if name == "" {
		writeError(w, http.StatusBadRequest, "не указан параметр metric")
		return
	}
	matchers := parseMatchers(r.URL.Query().Get("labels"))
	writeJSON(w, http.StatusOK, s.store.Latest(name, matchers))
}

func (s *Server) handleRange(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("metric")
	if name == "" {
		writeError(w, http.StatusBadRequest, "не указан параметр metric")
		return
	}
	window := 5 * time.Minute
	if raw := r.URL.Query().Get("range"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "некорректный range: "+err.Error())
			return
		}
		window = d
	}
	matchers := parseMatchers(r.URL.Query().Get("labels"))
	since := time.Now().Add(-window)
	writeJSON(w, http.StatusOK, s.store.Range(name, matchers, since))
}

func (s *Server) handleTargets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.manager.Targets())
}

func (s *Server) handleSelfMetrics(w http.ResponseWriter, r *http.Request) {
	seriesCount, sampleCount := s.store.Stats()
	targets := s.manager.Targets()
	upCount := 0
	for _, t := range targets {
		if t.Up {
			upCount++
		}
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "miniprom_series_total %d\n", seriesCount)
	fmt.Fprintf(w, "miniprom_samples_total %d\n", sampleCount)
	fmt.Fprintf(w, "miniprom_targets_total %d\n", len(targets))
	fmt.Fprintf(w, "miniprom_targets_up %d\n", upCount)
	fmt.Fprintf(w, "miniprom_uptime_seconds %.0f\n", time.Since(s.started).Seconds())
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

func parseMatchers(raw string) map[string]string {
	matchers := map[string]string{}
	if raw == "" {
		return matchers
	}
	for _, pair := range strings.Split(raw, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			matchers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return matchers
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Round(time.Microsecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
