package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
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
