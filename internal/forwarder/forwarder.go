package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultTimeout = 10 * time.Second

type Target struct {
	Type    string
	URL     string
	Method  string
	Headers map[string]string
	Timeout time.Duration
}

type EventEntry struct {
	Event map[string]interface{} `json:"event"`
	Bus   string                 `json:"bus"`
}

func Forward(targets []Target, entry EventEntry) []ForwardResult {
	if len(targets) == 0 {
		return nil
	}

	results := make([]ForwardResult, len(targets))
	for i, t := range targets {
		switch t.Type {
		case "http":
			results[i] = forwardHTTP(t, entry.Event)
		case "log":
			results[i] = ForwardResult{Success: true, TargetType: "log"}
		default:
			results[i] = ForwardResult{
				Success:    false,
				TargetType: t.Type,
				Error:      fmt.Sprintf("unknown target type: %s", t.Type),
			}
		}
	}
	return results
}

type ForwardResult struct {
	Success    bool   `json:"success"`
	TargetType string `json:"target_type"`
	TargetURL  string `json:"target_url,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

func forwardHTTP(t Target, event map[string]interface{}) ForwardResult {
	result := ForwardResult{
		TargetType: "http",
		TargetURL:  t.URL,
	}

	body, err := json.Marshal(event)
	if err != nil {
		result.Error = fmt.Sprintf("marshal event: %v", err)
		return result
	}

	method := t.Method
	if method == "" {
		method = http.MethodPost
	}

	timeout := t.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, t.URL, bytes.NewReader(body))
	if err != nil {
		result.Error = fmt.Sprintf("create request: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
	} else {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return result
}
