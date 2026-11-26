package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
)

//go:embed web/*
var webFiles embed.FS

type Process struct {
	PID    int     `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

type Stats struct {
	Timestamp      time.Time `json:"timestamp"`
	CPU            float64   `json:"cpu"`
	MemoryUsed     uint64    `json:"memory_used"`
	MemoryTotal    uint64    `json:"memory_total"`
	MemoryPercent  float64   `json:"memory_percent"`
	MemoryPressure string    `json:"memory_pressure"`
	SwapUsed       uint64    `json:"swap_used"`
	Battery        int       `json:"battery"`
	Uptime         string    `json:"uptime"`
	TopProcesses   []Process `json:"top_processes"`
	Self           Process   `json:"self"`
}

type UsageResponse struct {
	UpdatedAt   time.Time        `json:"updatedAt"`
	OpenCode    OpenCodeUsage    `json:"opencode"`
	CommandCode CommandCodeUsage `json:"commandCode"`
	Claude      ClaudeUsage      `json:"claude"`
	Antigravity AntigravityUsage `json:"antigravity"`
}

type UsageTokens struct {
	Input      int64 `json:"input"`
	Output     int64 `json:"output"`
	CacheRead  int64 `json:"cacheRead"`
	CacheWrite int64 `json:"cacheWrite"`
}

type ClaudeUsage struct {
	Error    string        `json:"error,omitempty"`
	Sessions int           `json:"sessions"`
	Messages int           `json:"messages"`
	Tokens   UsageTokens   `json:"tokens"`
	Models   []ClaudeModel `json:"models"`
}

type ClaudeModel struct {
	Model    string      `json:"model"`
	Messages int         `json:"messages"`
	Tokens   UsageTokens `json:"tokens"`
}

type AntigravityUsage struct {
	Error    string             `json:"error,omitempty"`
	Sessions int                `json:"sessions"`
	Steps    int                `json:"steps"`
	Tokens   UsageTokens        `json:"tokens"`
	Models   []AntigravityModel `json:"models"`
}

type AntigravityModel struct {
	Model  string      `json:"model"`
	Steps  int         `json:"steps"`
	Tokens UsageTokens `json:"tokens"`
}

type OpenCodeUsage struct {
	Error    string          `json:"error,omitempty"`
	Sessions int             `json:"sessions,omitempty"`
	Prompts  int             `json:"prompts,omitempty"`
	Steps    int             `json:"steps,omitempty"`
	Cost     float64         `json:"cost,omitempty"`
	Tokens   OpenCodeTokens  `json:"tokens,omitempty"`
	Models   []OpenCodeModel `json:"models,omitempty"`
	Range    json.RawMessage `json:"range,omitempty"`
}

type OpenCodeTokens struct {
	Input     int `json:"input"`
	Output    int `json:"output"`
	Reasoning int `json:"reasoning"`
	Cache     struct {
		Read  int `json:"read"`
		Write int `json:"write"`
	} `json:"cache"`
}

type OpenCodeModel struct {
	Model struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
		Variant    string `json:"variant"`
	} `json:"model"`
	Steps  int            `json:"steps"`
	Tokens OpenCodeTokens `json:"tokens"`
	Cost   float64        `json:"cost"`
}

type CommandCodeUsage struct {
	Error   string `json:"error,omitempty"`
	Account string `json:"account,omitempty"`
	PlanID  string `json:"planId,omitempty"`
	Status  string `json:"status,omitempty"`
	Credits struct {
		Remaining float64 `json:"remaining"`
		Monthly   float64 `json:"monthlyRemaining"`
		Purchased float64 `json:"purchased"`
		Free      float64 `json:"free"`
	} `json:"credits"`
	Windows struct {
		FiveHour map[string]any `json:"fiveHour"`
		Weekly   map[string]any `json:"weekly"`
	} `json:"windows"`
	Summary CommandCodeSummary `json:"summary"`
}

type CommandCodeSummary struct {
	Requests      int     `json:"requests"`
	RequestsToday int     `json:"requestsToday"`
	Cost          float64 `json:"cost"`
	CostToday     float64 `json:"costToday"`
	InputTokens   int64   `json:"inputTokens"`
	OutputTokens  int64   `json:"outputTokens"`
	TotalTokens   int64   `json:"totalTokens"`
	TodayTokens   int64   `json:"todayTokens"`
}

var statsCache struct {
	sync.RWMutex
	stats Stats
}

func main() {
	port := os.Getenv("SHISHIMON_PORT")
	if port == "" {
		port = os.Getenv("BROSTAT_PORT")
	}
	if port == "" {
		port = "8080"
	}
	token := randomToken()
	ip := localIP()
	url := fmt.Sprintf("http://%s:%s/?token=%s", ip, port, token)

	mux := http.NewServeMux()
	mux.HandleFunc("/", auth(token, dashboardHandler))
	mux.HandleFunc("/events", auth(token, eventsHandler))
	mux.HandleFunc("/api/usage", auth(token, usageHandler))
	mux.HandleFunc("/qr", auth(token, qrHandler(url)))

	server := &http.Server{Addr: ":" + port, Handler: mux}
	statsCache.stats = collectStats()
	go refreshStats()
	fmt.Printf("\nShishimon is running\n\n  URL: %s\n\n", url)
	if qr, err := qrcode.New(url, qrcode.Medium); err == nil {
		fmt.Println(qr.ToSmallString(false))
	} else {
		log.Printf("could not render terminal QR code: %v", err)
	}
	fmt.Println("Run bro-stop to stop Shishimon.")
	log.Fatal(server.ListenAndServe())
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "dashboard unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func usageHandler(w http.ResponseWriter, r *http.Request) {
	rangeName := r.URL.Query().Get("range")
	if !contains([]string{"today", "7d", "30d", "year", "all"}, rangeName) {
		rangeName = "today"
	}

	response := UsageResponse{UpdatedAt: time.Now()}
	openCode, err := collectOpenCodeUsage(rangeName)
	if err != nil {
		openCode.Error = err.Error()
	}
	response.OpenCode = openCode

	commandCode, err := collectCommandCodeUsage()
	if err != nil {
		commandCode.Error = err.Error()
	}
	response.CommandCode = commandCode

	claude, err := collectClaudeUsage(rangeName)
	if err != nil {
		claude.Error = err.Error()
	}
	response.Claude = claude

	antigravity, err := collectAntigravityUsage(rangeName)
	if err != nil {
		antigravity.Error = err.Error()
	}
	response.Antigravity = antigravity

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(response)
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func collectOpenCodeUsage(rangeName string) (OpenCodeUsage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return OpenCodeUsage{}, err
	}
	bin := os.Getenv("OPENCODE_BIN")
	if bin == "" {
		bin = filepath.Join(home, ".opencode", "bin", "opencode")
	}
	args := []string{"stats", "--models", "--cost", "--json"}
	switch rangeName {
	case "today":
		args = append(args, "--days", "0")
	case "7d":
		args = append(args, "--days", "7")
	case "30d":
		args = append(args, "--days", "30")
	case "year":
		args = append(args, "--year", strconv.Itoa(time.Now().Year()))
	default:
		args = append(args, "--all")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	output, err := cmd.Output()
	if err != nil {
		return OpenCodeUsage{}, fmt.Errorf("OpenCode stats unavailable: %w", err)
	}

	var result OpenCodeUsage
	if err := json.Unmarshal(output, &result); err != nil {
		return OpenCodeUsage{}, fmt.Errorf("could not parse OpenCode stats: %w", err)
	}
	return result, nil
}

func commandCodeKey() (string, error) {
	if key := os.Getenv("COMMAND_CODE_API_KEY"); key != "" {
		return key, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	authFile := os.Getenv("COMMAND_CODE_AUTH_FILE")
	if authFile == "" {
		authFile = filepath.Join(home, ".commandcode", "auth.json")
	}
	data, err := os.ReadFile(authFile)
	if err != nil {
		return "", fmt.Errorf("Command Code is not authenticated; run cmd login")
	}
	var auth struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(data, &auth); err != nil || auth.APIKey == "" {
		return "", fmt.Errorf("Command Code auth file is invalid")
	}
	return auth.APIKey, nil
}

func commandCodeRequest(pathname string, orgID string) (map[string]any, error) {
	key, err := commandCodeKey()
	if err != nil {
		return nil, err
	}
	baseURL := os.Getenv("COMMAND_CODE_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.commandcode.ai"
	}
	if orgID != "" {
		separator := "?"
		if strings.Contains(pathname, "?") {
			separator = "&"
		}
		pathname += separator + "orgId=" + url.QueryEscape(orgID)
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+pathname, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Command Code API request failed: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Command Code API %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("could not parse Command Code API: %w", err)
	}
	return result, nil
}

func collectCommandCodeUsage() (CommandCodeUsage, error) {
	whoami, err := commandCodeRequest("/alpha/whoami?limits=1", "")
	if err != nil {
		return CommandCodeUsage{}, err
	}
	orgID := ""
	if org, ok := whoami["org"].(map[string]any); ok {
		orgID, _ = org["id"].(string)
	}
	credits, err := commandCodeRequest("/alpha/billing/credits", orgID)
	if err != nil {
		return CommandCodeUsage{}, err
	}
	subscriptions, err := commandCodeRequest("/alpha/billing/subscriptions", orgID)
	if err != nil {
		return CommandCodeUsage{}, err
	}
	summary, err := commandCodeRequest("/alpha/usage/summary", orgID)
	if err != nil {
		return CommandCodeUsage{}, err
	}
	// The API expects UTC, so convert the local start of today to UTC.
	now := time.Now()
	localMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	today, err := commandCodeRequest("/alpha/usage/summary?since="+url.QueryEscape(localMidnight.UTC().Format("2006-01-02T15:04:05.000Z")), orgID)
	if err != nil {
		today = map[string]any{}
	}

	var result CommandCodeUsage
	if user, ok := whoami["user"].(map[string]any); ok {
		result.Account, _ = firstString(user, "userName", "name")
	}
	if data, ok := subscriptions["data"].(map[string]any); ok {
		result.PlanID, _ = data["planId"].(string)
		result.Status, _ = data["status"].(string)
	}
	creditPool, _ := credits["credits"].(map[string]any)
	monthly := number(creditPool["monthlyCredits"])
	purchased := number(creditPool["purchasedCredits"])
	free := number(creditPool["freeCredits"])
	result.Credits.Monthly = monthly
	result.Credits.Purchased = purchased
	result.Credits.Free = free
	result.Credits.Remaining = monthly + purchased + free
	if windows, ok := credits["windowLimits"].(map[string]any); ok {
		if five, ok := windows["fiveHour"].(map[string]any); ok {
			result.Windows.FiveHour = five
		}
		if weekly, ok := windows["weekly"].(map[string]any); ok {
			result.Windows.Weekly = weekly
		}
	}
	result.Summary.Requests = int(number(summary["totalCount"]))
	result.Summary.RequestsToday = int(number(today["totalCount"]))
	result.Summary.Cost = number(summary["totalCost"])
	result.Summary.CostToday = number(today["totalCost"])
	result.Summary.InputTokens = int64(number(summary["totalTokensIn"]))
	result.Summary.OutputTokens = int64(number(summary["totalTokensOut"]))
	result.Summary.TotalTokens = int64(number(summary["totalTokens"]))
	result.Summary.TodayTokens = int64(number(today["totalTokens"]))
	return result, nil
}

func number(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	default:
		return 0
	}
}

func firstString(values map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			return value, true
		}
	}
	return "", false
}

type usageEvent struct {
	Timestamp time.Time
	Session   string
	Model     string
	Tokens    UsageTokens
}

func (tokens *UsageTokens) add(other UsageTokens) {
	tokens.Input += other.Input
	tokens.Output += other.Output
	tokens.CacheRead += other.CacheRead
	tokens.CacheWrite += other.CacheWrite
}

func (tokens UsageTokens) total() int64 {
	return tokens.Input + tokens.Output + tokens.CacheRead + tokens.CacheWrite
}

const usageCacheTTL = 60 * time.Second

var claudeEventsCache struct {
	sync.Mutex
	loadedAt time.Time
	events   []usageEvent
	err      error
}

var antigravityEventsCache struct {
	sync.Mutex
	loadedAt time.Time
	events   []usageEvent
	err      error
}

func rangeCutoff(rangeName string) (time.Time, bool) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch rangeName {
	case "today":
		return startOfToday, true
	case "7d":
		return startOfToday.AddDate(0, 0, -6), true
	case "30d":
		return startOfToday.AddDate(0, 0, -29), true
	case "year":
		return time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location()), true
	default:
		return time.Time{}, false
	}
}

func claudeUsageEvents() ([]usageEvent, error) {
	claudeEventsCache.Lock()
	defer claudeEventsCache.Unlock()
	if time.Since(claudeEventsCache.loadedAt) < usageCacheTTL {
		return claudeEventsCache.events, claudeEventsCache.err
	}
	events, err := scanClaudeUsage()
	claudeEventsCache.events = events
	claudeEventsCache.err = err
	claudeEventsCache.loadedAt = time.Now()
	return events, err
}

func antigravityUsageEvents() ([]usageEvent, error) {
	antigravityEventsCache.Lock()
	defer antigravityEventsCache.Unlock()
	if time.Since(antigravityEventsCache.loadedAt) < usageCacheTTL {
		return antigravityEventsCache.events, antigravityEventsCache.err
	}
	events, err := scanAntigravityUsage()
	antigravityEventsCache.events = events
	antigravityEventsCache.err = err
	antigravityEventsCache.loadedAt = time.Now()
	return events, err
}

func collectClaudeUsage(rangeName string) (ClaudeUsage, error) {
	events, err := claudeUsageEvents()
	if err != nil {
		return ClaudeUsage{}, err
	}
	cutoff, bounded := rangeCutoff(rangeName)
	result := ClaudeUsage{Models: []ClaudeModel{}}
	sessions := map[string]struct{}{}
	models := map[string]*ClaudeModel{}
	for _, event := range events {
		if bounded && event.Timestamp.Before(cutoff) {
			continue
		}
		result.Messages++
		result.Tokens.add(event.Tokens)
		if event.Session != "" {
			sessions[event.Session] = struct{}{}
		}
		name := event.Model
		if name == "" || name == "<synthetic>" {
			name = "unknown"
		}
		row := models[name]
		if row == nil {
			row = &ClaudeModel{Model: name}
			models[name] = row
		}
		row.Messages++
		row.Tokens.add(event.Tokens)
	}
	result.Sessions = len(sessions)
	for _, row := range models {
		result.Models = append(result.Models, *row)
	}
	sort.Slice(result.Models, func(i, j int) bool {
		if result.Models[i].Tokens.total() == result.Models[j].Tokens.total() {
			return result.Models[i].Model < result.Models[j].Model
		}
		return result.Models[i].Tokens.total() > result.Models[j].Tokens.total()
	})
	return result, nil
}

func collectAntigravityUsage(rangeName string) (AntigravityUsage, error) {
	events, err := antigravityUsageEvents()
	if err != nil {
		return AntigravityUsage{}, err
	}
	cutoff, bounded := rangeCutoff(rangeName)
	result := AntigravityUsage{Models: []AntigravityModel{}}
	sessions := map[string]struct{}{}
	models := map[string]*AntigravityModel{}
	for _, event := range events {
		if bounded && event.Timestamp.Before(cutoff) {
			continue
		}
		result.Steps++
		result.Tokens.add(event.Tokens)
		if event.Session != "" {
			sessions[event.Session] = struct{}{}
		}
		name := event.Model
		if name == "" {
			name = "unknown"
		}
		row := models[name]
		if row == nil {
			row = &AntigravityModel{Model: name}
			models[name] = row
		}
		row.Steps++
		row.Tokens.add(event.Tokens)
	}
	result.Sessions = len(sessions)
	for _, row := range models {
		result.Models = append(result.Models, *row)
	}
	sort.Slice(result.Models, func(i, j int) bool {
		if result.Models[i].Tokens.total() == result.Models[j].Tokens.total() {
			return result.Models[i].Model < result.Models[j].Model
		}
		return result.Models[i].Tokens.total() > result.Models[j].Tokens.total()
	})
	return result, nil
}

func scanClaudeUsage() ([]usageEvent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		configDir = filepath.Join(home, ".claude")
	}
	projectsDir := filepath.Join(configDir, "projects")
	info, err := os.Stat(projectsDir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	var events []usageEvent
	seen := map[string]struct{}{}
	err = filepath.WalkDir(projectsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		events = append(events, scanClaudeFile(path, seen)...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func scanClaudeFile(path string, seen map[string]struct{}) []usageEvent {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var events []usageEvent
	fallbackSession := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 64*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"assistant"`)) || !bytes.Contains(line, []byte(`"usage"`)) {
			continue
		}
		var entry struct {
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
			SessionID string `json:"sessionId"`
			Message   struct {
				ID    string `json:"id"`
				Model string `json:"model"`
				Usage struct {
					InputTokens              int64 `json:"input_tokens"`
					OutputTokens             int64 `json:"output_tokens"`
					CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
					CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal(line, &entry); err != nil || entry.Type != "assistant" {
			continue
		}
		tokens := UsageTokens{
			Input:      entry.Message.Usage.InputTokens,
			Output:     entry.Message.Usage.OutputTokens,
			CacheRead:  entry.Message.Usage.CacheReadInputTokens,
			CacheWrite: entry.Message.Usage.CacheCreationInputTokens,
		}
		if tokens.total() == 0 {
			continue
		}
		key := entry.Message.ID
		if key == "" {
			key = fmt.Sprintf("%s:%d", path, lineNumber)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		session := entry.SessionID
		if session == "" {
			session = fallbackSession
		}
		timestamp, _ := time.Parse(time.RFC3339Nano, entry.Timestamp)
		events = append(events, usageEvent{
			Timestamp: timestamp,
			Session:   session,
			Model:     entry.Message.Model,
			Tokens:    tokens,
		})
	}
	return events
}

func scanAntigravityUsage() ([]usageEvent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	geminiDir := os.Getenv("GEMINI_DIR")
	if geminiDir == "" {
		geminiDir = filepath.Join(home, ".gemini")
	}

	var events []usageEvent
	seen := map[string]struct{}{}
	for _, app := range []string{"antigravity", "antigravity-cli", "antigravity-ide"} {
		pattern := filepath.Join(geminiDir, app, "brain", "*", ".system_generated", "logs", "transcript_full.jsonl")
		files, _ := filepath.Glob(pattern)
		for _, path := range files {
			conversation := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
			events = append(events, scanAntigravityFile(path, conversation, seen)...)
		}
	}
	return events, nil
}

func scanAntigravityFile(path string, conversation string, seen map[string]struct{}) []usageEvent {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var events []usageEvent
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 64*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"input_tokens"`)) {
			continue
		}
		var entry struct {
			StepIndex       int    `json:"step_index"`
			Source          string `json:"source"`
			CreatedAt       string `json:"created_at"`
			InputTokens     int64  `json:"input_tokens"`
			CacheReadTokens int64  `json:"cache_read_tokens"`
			OutputTokens    int64  `json:"output_tokens"`
			Model           string `json:"model"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Source != "" && entry.Source != "MODEL" {
			continue
		}
		tokens := UsageTokens{
			Input:     entry.InputTokens,
			Output:    entry.OutputTokens,
			CacheRead: entry.CacheReadTokens,
		}
		if tokens.total() == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%d:%s:%d:%d:%d", conversation, entry.StepIndex, entry.CreatedAt, tokens.Input, tokens.Output, tokens.CacheRead)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		timestamp, _ := time.Parse(time.RFC3339Nano, entry.CreatedAt)
		events = append(events, usageEvent{
			Timestamp: timestamp,
			Session:   conversation,
			Model:     entry.Model,
			Tokens:    tokens,
		})
	}
	return events
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		statsCache.RLock()
		stats := statsCache.stats
		statsCache.RUnlock()
		payload, err := json.Marshal(stats)
		if err != nil {
			http.Error(w, "could not encode stats", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func refreshStats() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		stats := collectStats()
		statsCache.Lock()
		statsCache.stats = stats
		statsCache.Unlock()
	}
}

func qrHandler(url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		png, err := qrcode.Encode(url, qrcode.Medium, 512)
		if err != nil {
			http.Error(w, "could not create QR code", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}
}

func auth(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func collectStats() Stats {
	total, used := memoryStats()
	return Stats{
		Timestamp:      time.Now(),
		CPU:            cpuUsage(),
		MemoryUsed:     used,
		MemoryTotal:    total,
		MemoryPercent:  percent(used, total),
		MemoryPressure: memoryPressure(),
		SwapUsed:       swapUsed(),
		Battery:        batteryPercent(),
		Uptime:         uptime(),
		TopProcesses:   processes(),
		Self:           processUsage(os.Getpid()),
	}
}

func command(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func cpuUsage() float64 {
	for _, line := range strings.Split(command("top", "-l", "1", "-n", "0"), "\n") {
		if !strings.Contains(line, "CPU usage:") {
			continue
		}
		fields := strings.Fields(line)
		var user, system float64
		for i, field := range fields {
			if i == 0 || !strings.HasSuffix(field, "%") || i+1 >= len(fields) {
				continue
			}
			value, _ := strconv.ParseFloat(strings.TrimSuffix(field, "%"), 64)
			if fields[i+1] == "user," {
				user = value
			}
			if fields[i+1] == "sys," {
				system = value
			}
		}
		return user + system
	}
	return 0
}

func memoryStats() (uint64, uint64) {
	physical, _ := strconv.ParseUint(strings.TrimSpace(command("sysctl", "-n", "hw.memsize")), 10, 64)
	if physical == 0 {
		return 0, 0
	}
	pageSize := uint64(4096)
	var active, wired uint64
	for _, line := range strings.Split(command("vm_stat"), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !strings.HasPrefix(fields[0], "Pages") {
			continue
		}
		value, err := strconv.ParseUint(strings.Trim(fields[len(fields)-1], "."), 10, 64)
		if err != nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "Pages active"):
			active = value
		case strings.HasPrefix(line, "Pages wired down"):
			wired = value
		}
	}
	compressor := uint64(0)
	for _, line := range strings.Split(command("top", "-l", "1", "-n", "0"), "\n") {
		if strings.HasPrefix(line, "PhysMem:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if strings.HasPrefix(field, "compressor") && i > 0 {
					compressor = parseSize(fields[i-1])
				}
			}
		}
	}
	used := active*pageSize + wired*pageSize + compressor
	if used > physical {
		used = physical
	}
	return physical, used
}

func memoryPressure() string {
	out := command("sysctl", "-n", "vm.memory_pressure")
	if strings.TrimSpace(out) == "1" {
		return "High"
	}
	return "Normal"
}

func swapUsed() uint64 {
	fields := strings.Fields(command("sysctl", "-n", "vm.swapusage"))
	for i, field := range fields {
		if field == "used" && i+1 < len(fields) {
			return parseSize(fields[i+1])
		}
	}
	return 0
}

func batteryPercent() int {
	for _, field := range strings.Fields(command("pmset", "-g", "batt")) {
		if strings.HasSuffix(field, "%;") {
			value, err := strconv.Atoi(strings.TrimSuffix(field, "%;"))
			if err == nil {
				return value
			}
		}
	}
	return -1
}

func processes() []Process {
	var result []Process
	for _, line := range strings.Split(command("ps", "-Aceo", "pid=,pcpu=,pmem=,rss=,comm="), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		cpu, cpuErr := strconv.ParseFloat(fields[1], 64)
		rss, rssErr := strconv.ParseFloat(fields[3], 64)
		if err != nil || cpuErr != nil || rssErr != nil {
			continue
		}
		result = append(result, Process{PID: pid, Name: strings.Join(fields[4:], " "), CPU: cpu, Memory: rss * 1024})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CPU > result[j].CPU })
	if len(result) > 8 {
		result = result[:8]
	}
	return result
}

func processUsage(pid int) Process {
	fields := strings.Fields(command("ps", "-p", strconv.Itoa(pid), "-o", "pcpu=,rss="))
	if len(fields) < 2 {
		return Process{PID: pid, Name: "shishimon"}
	}
	cpu, _ := strconv.ParseFloat(fields[0], 64)
	rss, _ := strconv.ParseFloat(fields[1], 64)
	return Process{PID: pid, Name: "shishimon", CPU: cpu, Memory: rss * 1024}
}

func uptime() string {
	output := command("sysctl", "-n", "kern.boottime")
	secIndex := strings.Index(output, "sec = ")
	var seconds int64
	if secIndex >= 0 {
		value := strings.TrimSuffix(strings.Fields(output[secIndex+len("sec = "):])[0], ",")
		seconds, _ = strconv.ParseInt(value, 10, 64)
	}
	if seconds == 0 {
		return "Unknown"
	}
	return time.Since(time.Unix(seconds, 0)).Round(time.Minute).String()
}

func localIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err == nil && ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return "localhost"
}

func randomToken() string {
	var bytes [4]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		log.Fatalf("could not create access token: %v", err)
	}
	number := (uint32(bytes[0])<<24 | uint32(bytes[1])<<16 | uint32(bytes[2])<<8 | uint32(bytes[3])) % 100000
	return fmt.Sprintf("%05d", number)
}

func parseSize(value string) uint64 {
	multiplier := float64(1)
	switch strings.ToUpper(value[len(value)-1:]) {
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "M":
		multiplier = 1024 * 1024
	case "K":
		multiplier = 1024
	}
	number, _ := strconv.ParseFloat(strings.TrimRight(value, "kKmMgGtT"), 64)
	return uint64(number * multiplier)
}

func percent(value, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) / float64(total) * 100
}
