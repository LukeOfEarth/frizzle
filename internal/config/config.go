package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Port int      `json:"port"`
	Buses []Bus   `json:"buses"`
}

type Bus struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

type Rule struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Pattern     map[string]interface{} `json:"pattern"`
	Targets     []Target            `json:"targets"`
}

type Target struct {
	Type    string            `json:"type"`
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout string            `json:"timeout,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if cfg.Port <= 0 {
		cfg.Port = 4000
	}

	for i := range cfg.Buses {
		if cfg.Buses[i].Name == "" {
			cfg.Buses[i].Name = "default"
		}
		for j := range cfg.Buses[i].Rules {
			if cfg.Buses[i].Rules[j].Pattern == nil {
				cfg.Buses[i].Rules[j].Pattern = map[string]interface{}{}
			}
		}
	}

	return &cfg, nil
}

func SimpleConfig(port int, forwardURL string) *Config {
	return &Config{
		Port: port,
		Buses: []Bus{
			{
				Name: "default",
				Rules: []Rule{
					{
						Name:    "catch-all",
						Description: "Forward all events to downstream and log to console",
						Pattern: map[string]interface{}{},
						Targets: buildSimpleTargets(forwardURL),
					},
				},
			},
		},
	}
}

func buildSimpleTargets(url string) []Target {
	targets := []Target{
		{Type: "log"},
	}
	if url != "" {
		targets = append(targets, Target{
			Type:   "http",
			URL:    url,
			Method: "POST",
		})
	}
	return targets
}
