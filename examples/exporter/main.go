package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type state struct {
	mu          sync.Mutex
	requests    map[string]int
	temperature float64
	inflight    int
}

func main() {
	addr := flag.String("addr", ":9100", "адрес для /metrics")
	flag.Parse()

	s := &state{
		requests:    map[string]int{"200": 0, "404": 0, "500": 0},
		temperature: 21.5,
	}

	go s.simulate()

	http.HandleFunc("/metrics", s.metrics)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "demo exporter, метрики на /metrics")
	})

	log.Printf("demo exporter слушает на %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *state) simulate() {
	codes := []string{"200", "200", "200", "200", "404", "500"}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		s.requests[codes[rand.Intn(len(codes))]]++
		s.temperature += (rand.Float64() - 0.5) * 0.8
		s.temperature = math.Max(18, math.Min(26, s.temperature))
		s.inflight = rand.Intn(12)
		s.mu.Unlock()
	}
}

func (s *state) metrics(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintln(w, "# HELP http_requests_total Общее число обработанных запросов.")
	fmt.Fprintln(w, "# TYPE http_requests_total counter")
	for code, count := range s.requests {
		fmt.Fprintf(w, "http_requests_total{code=%q} %d\n", code, count)
	}
	fmt.Fprintln(w, "# HELP room_temperature_celsius Температура в комнате.")
	fmt.Fprintln(w, "# TYPE room_temperature_celsius gauge")
	fmt.Fprintf(w, "room_temperature_celsius %.2f\n", s.temperature)
	fmt.Fprintln(w, "# HELP http_requests_in_flight Запросы в обработке прямо сейчас.")
	fmt.Fprintln(w, "# TYPE http_requests_in_flight gauge")
	fmt.Fprintf(w, "http_requests_in_flight %d\n", s.inflight)
}
