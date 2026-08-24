package storage

import (
	"testing"
	"time"
)

func TestAddAndLatest(t *testing.T) {
	s := New(time.Hour)
	now := time.Now()
	s.Add("cpu", map[string]string{"host": "a"}, 1, now)
	s.Add("cpu", map[string]string{"host": "a"}, 2, now.Add(time.Second))
	s.Add("cpu", map[string]string{"host": "b"}, 9, now)

	latest := s.Latest("cpu", nil)
	if len(latest) != 2 {
		t.Fatalf("ожидалось 2 серии, получено %d", len(latest))
	}

	filtered := s.Latest("cpu", map[string]string{"host": "a"})
	if len(filtered) != 1 {
		t.Fatalf("ожидалась 1 серия после фильтра, получено %d", len(filtered))
	}
	if v := filtered[0].Samples[0].V; v != 2 {
		t.Errorf("ожидалось последнее значение 2, получено %v", v)
	}
}

func TestRange(t *testing.T) {
	s := New(time.Hour)
	now := time.Now()
	s.Add("m", nil, 1, now.Add(-10*time.Minute))
	s.Add("m", nil, 2, now.Add(-1*time.Minute))

	res := s.Range("m", nil, now.Add(-5*time.Minute))
	if len(res) != 1 || len(res[0].Samples) != 1 {
		t.Fatalf("ожидался 1 сэмпл в окне, получено %+v", res)
	}
}

func TestCompaction(t *testing.T) {
	s := New(30 * time.Minute)
	now := time.Now()
	s.Add("old", nil, 1, now.Add(-2*time.Hour))
	s.Add("fresh", nil, 1, now)

	s.Compact(now)

	if got := s.Latest("old", nil); len(got) != 0 {
		t.Errorf("старая серия должна быть удалена, осталось %d", len(got))
	}
	if got := s.Latest("fresh", nil); len(got) != 1 {
		t.Errorf("свежая серия должна остаться")
	}
}
