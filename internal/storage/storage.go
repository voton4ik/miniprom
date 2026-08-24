package storage

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type Sample struct {
	T time.Time `json:"t"`
	V float64   `json:"v"`
}

type Series struct {
	Name    string            `json:"name"`
	Labels  map[string]string `json:"labels"`
	Samples []Sample          `json:"samples"`
}

type Storage struct {
	mu        sync.RWMutex
	series    map[string]*Series
	retention time.Duration
}

func New(retention time.Duration) *Storage {
	return &Storage{
		series:    map[string]*Series{},
		retention: retention,
	}
}

func (s *Storage) Add(name string, labels map[string]string, value float64, t time.Time) {
	key := seriesKey(name, labels)
	s.mu.Lock()
	defer s.mu.Unlock()
	ser, ok := s.series[key]
	if !ok {
		cp := make(map[string]string, len(labels))
		for k, v := range labels {
			cp[k] = v
		}
		ser = &Series{Name: name, Labels: cp}
		s.series[key] = ser
	}
	ser.Samples = append(ser.Samples, Sample{T: t, V: value})
}

func (s *Storage) Latest(name string, matchers map[string]string) []Series {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Series
	for _, ser := range s.series {
		if ser.Name != name || !matches(ser.Labels, matchers) || len(ser.Samples) == 0 {
			continue
		}
		last := ser.Samples[len(ser.Samples)-1]
		out = append(out, Series{
			Name:    ser.Name,
			Labels:  cloneLabels(ser.Labels),
			Samples: []Sample{last},
		})
	}
	sortSeries(out)
	return out
}

func (s *Storage) Range(name string, matchers map[string]string, since time.Time) []Series {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Series
	for _, ser := range s.series {
		if ser.Name != name || !matches(ser.Labels, matchers) {
			continue
		}
		var picked []Sample
		for _, smp := range ser.Samples {
			if !smp.T.Before(since) {
				picked = append(picked, smp)
			}
		}
		if len(picked) == 0 {
			continue
		}
		out = append(out, Series{
			Name:    ser.Name,
			Labels:  cloneLabels(ser.Labels),
			Samples: picked,
		})
	}
	sortSeries(out)
	return out
}

func (s *Storage) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]struct{}{}
	for _, ser := range s.series {
		seen[ser.Name] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (s *Storage) Stats() (seriesCount int, sampleCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ser := range s.series {
		seriesCount++
		sampleCount += len(ser.Samples)
	}
	return
}

func (s *Storage) Compact(now time.Time) {
	cutoff := now.Add(-s.retention)
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, ser := range s.series {
		idx := 0
		for idx < len(ser.Samples) && ser.Samples[idx].T.Before(cutoff) {
			idx++
		}
		if idx > 0 {
			ser.Samples = append(ser.Samples[:0], ser.Samples[idx:]...)
		}
		if len(ser.Samples) == 0 {
			delete(s.series, key)
		}
	}
}

func (s *Storage) RunCompaction(stop <-chan struct{}, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			s.Compact(now)
		}
	}
}

func matches(labels, matchers map[string]string) bool {
	for k, v := range matchers {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func cloneLabels(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sortSeries(list []Series) {
	sort.Slice(list, func(i, j int) bool {
		return seriesKey(list[i].Name, list[i].Labels) < seriesKey(list[j].Name, list[j].Labels)
	})
}

func seriesKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(labels[k])
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}
