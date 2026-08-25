package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Target struct {
	Job string `json:"job"`
	URL string `json:"url"`
}

type Config struct {
	ListenAddr     string   `json:"listen_addr"`
	ScrapeInterval Duration `json:"scrape_interval"`
	Retention      Duration `json:"retention"`
	Timeout        Duration `json:"timeout"`
	Targets        []Target `json:"targets"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := json.Unmarshal(raw, cfg); err != nil {
		return nil, err
	}
	applyDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":9099"
	}
	if cfg.ScrapeInterval.Duration == 0 {
		cfg.ScrapeInterval.Duration = 15 * time.Second
	}
	if cfg.Retention.Duration == 0 {
		cfg.Retention.Duration = time.Hour
	}
	if cfg.Timeout.Duration == 0 {
		cfg.Timeout.Duration = 10 * time.Second
	}
}

func validate(cfg *Config) error {
	if len(cfg.Targets) == 0 {
		return fmt.Errorf("нужно указать хотя бы одну цель в targets")
	}
	for i, t := range cfg.Targets {
		if t.URL == "" {
			return fmt.Errorf("targets[%d]: пустой url", i)
		}
		if t.Job == "" {
			return fmt.Errorf("targets[%d]: пустой job", i)
		}
	}
	if cfg.Timeout.Duration >= cfg.ScrapeInterval.Duration {
		return fmt.Errorf("timeout должен быть меньше scrape_interval")
	}
	return nil
}
