package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LukeOfEarth/frizzle/internal/config"
	"github.com/LukeOfEarth/frizzle/internal/forwarder"
	"github.com/LukeOfEarth/frizzle/internal/matcher"
)

type Server struct {
	cfg    *config.Config
	mux    *http.ServeMux
	logger *log.Logger
}

func New(cfg *config.Config, logger *log.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		mux:    http.NewServeMux(),
		logger: logger,
	}
	s.mux.HandleFunc("/", s.handleEvents)
	return s
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.logger.Printf("Event bus listening on %s", addr)
	s.logger.Printf("  SDK endpoint: http://localhost%s (set as EventBridgeClient endpoint)", addr)
	s.logger.Printf("  Curl endpoint: http://localhost%s/events/{bus-name}", addr)
	s.logger.Printf("  Configured buses: %s", strings.Join(s.busNames(), ", "))
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) busNames() []string {
	names := make([]string, len(s.cfg.Buses))
	for i, b := range s.cfg.Buses {
		names[i] = b.Name
	}
	return names
}

type sdkEntry struct {
	Source       string   `json:"Source"`
	DetailType   string   `json:"DetailType"`
	Detail       string   `json:"Detail"`
	EventBusName string   `json:"EventBusName"`
	Time         string   `json:"Time"`
	Resources    []string `json:"Resources"`
	ID           string   `json:"Id"`
	Version      string   `json:"Version"`
	Account      string   `json:"Account"`
	Region       string   `json:"Region"`
}

type sdkRequest struct {
	Entries []sdkEntry `json:"Entries"`
}

type simpleRequest struct {
	Entries []map[string]interface{} `json:"Entries"`
}

type putEventsResponse struct {
	Entries          []putEventsEntry `json:"Entries"`
	FailedEntryCount int              `json:"FailedEntryCount"`
}

type putEventsEntry struct {
	EventID string `json:"EventId"`
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	target := r.Header.Get("X-Amz-Target")
	if target == "AWSEvents.PutEvents" {
		s.handleSDK(w, body)
		return
	}

	s.handleSimple(w, r.URL.Path, body)
}

func (s *Server) handleSDK(w http.ResponseWriter, body []byte) {
	var req sdkRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeSDKError(w, "InvalidRequest", err.Error())
		return
	}

	resp := putEventsResponse{}

	for _, entry := range req.Entries {
		busName := entry.EventBusName
		if busName == "" {
			busName = s.defaultBusName()
		}

		normalized := normalizeSDKEntry(entry)
		id := s.processEntry(busName, normalized)
		resp.Entries = append(resp.Entries, putEventsEntry{EventID: id})
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSimple(w http.ResponseWriter, path string, body []byte) {
	busName := s.resolveBusName(path)

	var req simpleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp := putEventsResponse{}
	for _, entry := range req.Entries {
		id := s.processEntry(busName, entry)
		resp.Entries = append(resp.Entries, putEventsEntry{EventID: id})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func normalizeSDKEntry(e sdkEntry) map[string]interface{} {
	entry := map[string]interface{}{
		"source":      e.Source,
		"detail-type": e.DetailType,
	}

	if e.Time != "" {
		entry["time"] = e.Time
	}
	if len(e.Resources) > 0 {
		resources := make([]interface{}, len(e.Resources))
		for i, r := range e.Resources {
			resources[i] = r
		}
		entry["resources"] = resources
	}
	if e.ID != "" {
		entry["id"] = e.ID
	}
	if e.Version != "" {
		entry["version"] = e.Version
	}
	if e.Account != "" {
		entry["account"] = e.Account
	}
	if e.Region != "" {
		entry["region"] = e.Region
	}

	if e.Detail != "" {
		var detail interface{}
		if err := json.Unmarshal([]byte(e.Detail), &detail); err == nil {
			entry["detail"] = detail
		} else {
			entry["detail"] = e.Detail
		}
	}

	return entry
}

func (s *Server) defaultBusName() string {
	if len(s.cfg.Buses) > 0 {
		return s.cfg.Buses[0].Name
	}
	return "default"
}

func writeSDKError(w http.ResponseWriter, typ, message string) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{
		"__type":  typ,
		"Message": message,
	})
}

func (s *Server) resolveBusName(path string) string {
	path = strings.TrimPrefix(path, "/events/")
	path = strings.Trim(path, "/")

	if path == "" || path == "events" {
		return s.defaultBusName()
	}

	for _, b := range s.cfg.Buses {
		if b.Name == path {
			return b.Name
		}
	}

	return s.defaultBusName()
}

func (s *Server) processEntry(busName string, entry map[string]interface{}) string {
	eventID := generateEventID(entry)
	l := s.logger

	l.Printf("--- EVENT %s ---", eventID)
	l.Printf("  Bus:    %s", busName)
	if src, ok := entry["source"]; ok {
		l.Printf("  Source: %v", src)
	}
	if dt, ok := entry["detail-type"]; ok {
		l.Printf("  Detail: %v", dt)
	}
	if detail, ok := entry["detail"]; ok {
		detailJSON, _ := json.MarshalIndent(detail, "  ", "  ")
		l.Printf("  Payload:\n  %s", string(detailJSON))
	}

	var allTargets []forwarder.Target
	for _, bus := range s.cfg.Buses {
		if bus.Name != busName {
			continue
		}
		for _, rule := range bus.Rules {
			if matcher.Matches(entry, rule.Pattern) {
				l.Printf("  Matched rule: %s", rule.Name)
				for _, t := range rule.Targets {
					d, _ := time.ParseDuration(t.Timeout)
					allTargets = append(allTargets, forwarder.Target{
						Type:    t.Type,
						URL:     t.URL,
						Method:  t.Method,
						Headers: t.Headers,
						Timeout: d,
					})
				}
			}
		}
	}

	if len(allTargets) == 0 {
		l.Printf("  No rules matched — event not forwarded")
		return eventID
	}

	var wg sync.WaitGroup
	results := make([]forwarder.ForwardResult, len(allTargets))
	for i, t := range allTargets {
		wg.Add(1)
		go func(idx int, target forwarder.Target) {
			defer wg.Done()
			results[idx] = s.forwardTo(target, entry)
		}(i, t)
	}
	wg.Wait()

	for _, r := range results {
		switch {
		case r.Success && r.TargetType == "log":
			l.Printf("  Logged to console")
		case r.Success:
			l.Printf("  Forwarded to %s → OK (HTTP %d)", r.TargetURL, r.StatusCode)
		default:
			l.Printf("  Forward to %s FAILED: %s", r.TargetURL, r.Error)
		}
	}

	return eventID
}

func (s *Server) forwardTo(t forwarder.Target, event map[string]interface{}) forwarder.ForwardResult {
	switch t.Type {
	case "http":
		return s.httpForward(t, event)
	case "log":
		return forwarder.ForwardResult{Success: true, TargetType: "log"}
	default:
		return forwarder.ForwardResult{Success: false, TargetType: t.Type, Error: fmt.Sprintf("unknown target type: %s", t.Type)}
	}
}

func (s *Server) httpForward(t forwarder.Target, event map[string]interface{}) forwarder.ForwardResult {
	return forwarder.Forward([]forwarder.Target{t}, forwarder.EventEntry{Event: event})[0]
}

func generateEventID(entry map[string]interface{}) string {
	if id, ok := entry["id"]; ok {
		if s, ok := id.(string); ok && s != "" {
			return s
		}
	}
	return fmt.Sprintf("local-%d", time.Now().UnixNano())
}
