package parser

import (
	"math"
	"strings"
	"testing"
)

func TestParseSimple(t *testing.T) {
	input := `
# HELP http_requests_total total
# TYPE http_requests_total counter
http_requests_total{code="200",method="get"} 1027
http_requests_total{code="500"} 3
room_temperature_celsius 21.4
`
	metrics, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(metrics) != 3 {
		t.Fatalf("ожидалось 3 метрики, получено %d", len(metrics))
	}
	first := metrics[0]
	if first.Name != "http_requests_total" {
		t.Errorf("имя: %q", first.Name)
	}
	if first.Labels["code"] != "200" || first.Labels["method"] != "get" {
		t.Errorf("метки разобраны неверно: %v", first.Labels)
	}
	if first.Value != 1027 {
		t.Errorf("значение: %v", first.Value)
	}
	last := metrics[2]
	if len(last.Labels) != 0 || last.Value != 21.4 {
		t.Errorf("метрика без меток разобрана неверно: %+v", last)
	}
}

func TestParseSpecialValues(t *testing.T) {
	metrics, err := Parse(strings.NewReader("a_metric +Inf\nb_metric NaN\n"))
	if err != nil {
		t.Fatalf("ошибка: %v", err)
	}
	if !math.IsInf(metrics[0].Value, 1) {
		t.Errorf("ожидался +Inf, получено %v", metrics[0].Value)
	}
	if !math.IsNaN(metrics[1].Value) {
		t.Errorf("ожидался NaN, получено %v", metrics[1].Value)
	}
}

func TestParseEscapedLabel(t *testing.T) {
	metrics, err := Parse(strings.NewReader(`path{route="a\"b"} 1`))
	if err != nil {
		t.Fatalf("ошибка: %v", err)
	}
	if metrics[0].Labels["route"] != `a"b` {
		t.Errorf("экранирование не сработало: %q", metrics[0].Labels["route"])
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse(strings.NewReader("broken_metric")); err == nil {
		t.Fatal("ожидалась ошибка для строки без значения")
	}
}

func TestParseEmptyAndComments(t *testing.T) {
	input := "# HELP foo bar\n# TYPE foo counter\n\n   \n"
	metrics, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ошибка: %v", err)
	}
	if len(metrics) != 0 {
		t.Fatalf("ожидалось 0 метрик, получено %d", len(metrics))
	}
}
