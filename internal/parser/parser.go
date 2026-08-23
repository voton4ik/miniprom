package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Metric struct {
	Name   string
	Labels map[string]string
	Value  float64
}

func Parse(r io.Reader) ([]Metric, error) {
	var out []Metric
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		m, err := parseLine(text)
		if err != nil {
			return nil, fmt.Errorf("строка %d: %w", line, err)
		}
		out = append(out, m)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseLine(text string) (Metric, error) {
	var m Metric
	m.Labels = map[string]string{}

	brace := strings.IndexByte(text, '{')
	if brace == -1 {
		fields := strings.Fields(text)
		if len(fields) < 2 {
			return m, fmt.Errorf("ожидалось имя и значение")
		}
		m.Name = fields[0]
		v, err := parseValue(fields[1])
		if err != nil {
			return m, err
		}
		m.Value = v
		return m, nil
	}

	m.Name = strings.TrimSpace(text[:brace])
	if m.Name == "" {
		return m, fmt.Errorf("пустое имя метрики")
	}

	end := strings.IndexByte(text[brace:], '}')
	if end == -1 {
		return m, fmt.Errorf("не закрыта скобка меток")
	}
	end += brace

	labels, err := parseLabels(text[brace+1 : end])
	if err != nil {
		return m, err
	}
	m.Labels = labels

	rest := strings.Fields(strings.TrimSpace(text[end+1:]))
	if len(rest) < 1 {
		return m, fmt.Errorf("отсутствует значение")
	}
	v, err := parseValue(rest[0])
	if err != nil {
		return m, err
	}
	m.Value = v
	return m, nil
}

func parseLabels(s string) (map[string]string, error) {
	labels := map[string]string{}
	s = strings.TrimSpace(s)
	if s == "" {
		return labels, nil
	}
	i := 0
	for i < len(s) {
		eq := strings.IndexByte(s[i:], '=')
		if eq == -1 {
			return nil, fmt.Errorf("метка без значения")
		}
		key := strings.TrimSpace(s[i : i+eq])
		i += eq + 1
		if i >= len(s) || s[i] != '"' {
			return nil, fmt.Errorf("значение метки %q должно быть в кавычках", key)
		}
		i++
		var val strings.Builder
		for i < len(s) {
			c := s[i]
			if c == '\\' && i+1 < len(s) {
				next := s[i+1]
				switch next {
				case 'n':
					val.WriteByte('\n')
				case '"':
					val.WriteByte('"')
				case '\\':
					val.WriteByte('\\')
				default:
					val.WriteByte(next)
				}
				i += 2
				continue
			}
			if c == '"' {
				i++
				break
			}
			val.WriteByte(c)
			i++
		}
		labels[key] = val.String()
		for i < len(s) && (s[i] == ',' || s[i] == ' ') {
			i++
		}
	}
	return labels, nil
}

func parseValue(s string) (float64, error) {
	switch s {
	case "+Inf":
		return strconv.ParseFloat("+Inf", 64)
	case "-Inf":
		return strconv.ParseFloat("-Inf", 64)
	case "NaN":
		return strconv.ParseFloat("NaN", 64)
	}
	return strconv.ParseFloat(s, 64)
}
