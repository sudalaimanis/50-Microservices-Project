// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// Package kit is the tiny shared runtime every RailShack Go microservice uses.
// It gives each service the same production basics: health and readiness
// probes, Prometheus metrics, structured logs, request-id propagation,
// service discovery, downstream calls with timeouts, and a trace header that
// lets the UI light up every service a request touched.
package kit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Brand identifies the project in every response, log line and metric.
const (
	Brand     = "RailShack by DevOps Shack"
	BrandURL  = "https://devopsshack.com"
	YouTube   = "https://www.youtube.com/@devopsshack"
	SourceURL = "https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack"
	PoweredBy = "RailShack by DevOps Shack (devopsshack.com)"
)

type Service struct {
	Name     string
	Version  string
	Lang     string
	mux      *http.ServeMux
	ports    map[string]int
	work     int
	latency  int
	failRate float64
	started  time.Time
	mu       sync.Mutex
	reqs     map[string]int64
	durSum   float64
	durCount int64
	client   *http.Client
}

type Ctx struct {
	W     http.ResponseWriter
	R     *http.Request
	S     *Service
	trace []string
	code  int
}

func envInt(k string, d int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return d
}

// New creates a service. Configuration comes only from environment variables
// so the same image runs locally, in Docker Compose and on Kubernetes.
func New(name string) *Service {
	s := &Service{
		Name: name, Lang: "go", Version: getenv("SERVICE_VERSION", "1.0.0"),
		mux: http.NewServeMux(), work: envInt("WORK_FACTOR", 0), latency: envInt("LATENCY_MS", 0),
		started: time.Now(), reqs: map[string]int64{},
		client: &http.Client{Timeout: time.Duration(envInt("CALL_TIMEOUT_MS", 3000)) * time.Millisecond},
	}
	if f, err := strconv.ParseFloat(os.Getenv("FAIL_RATE"), 64); err == nil {
		s.failRate = f
	}
	if pf := os.Getenv("PORTS_FILE"); pf != "" {
		s.ports = loadPorts(pf)
	}
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ready")) })
	s.mux.HandleFunc("GET /metrics", s.metrics)
	s.Handle("GET /api/"+name+"/info", func(c *Ctx) { c.JSON(200, s.Info()) })
	return s
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func loadPorts(path string) map[string]int {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("cannot read PORTS_FILE: %v", err)
	}
	var list []struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}
	json.Unmarshal(b, &list)
	m := map[string]int{}
	for _, e := range list {
		m[e.Name] = e.Port
	}
	return m
}

// Pod returns the pod (or host) name, so the UI can show which replica answered.
func Pod() string {
	if p := os.Getenv("POD_NAME"); p != "" {
		return p
	}
	h, _ := os.Hostname()
	return h
}

func (s *Service) Info() map[string]any {
	return map[string]any{"service": s.Name, "version": s.Version, "language": s.Lang, "pod": Pod(),
		"uptimeSeconds": int(time.Since(s.started).Seconds()),
		"project":       Brand, "website": BrandURL, "youtube": YouTube, "source": SourceURL, "license": "Apache-2.0"}
}

// URL resolves another service: local port map, explicit override, or Kubernetes DNS.
func (s *Service) URL(svc string) string {
	if p, ok := s.ports[svc]; ok {
		return fmt.Sprintf("http://127.0.0.1:%d", p)
	}
	if v := os.Getenv("SVC_" + strings.ToUpper(strings.ReplaceAll(svc, "-", "_")) + "_URL"); v != "" {
		return v
	}
	return "http://" + svc + ":8080"
}

// Handle registers a route ("METHOD /path/{param}") with metrics, logs and chaos settings applied.
func (s *Service) Handle(pattern string, fn func(*Ctx)) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "X-Trace, X-Pod")
		w.Header().Set("X-Pod", Pod())
		w.Header().Set("X-Powered-By", PoweredBy)
		c := &Ctx{W: w, R: r, S: s, code: 200}
		if s.latency > 0 {
			time.Sleep(time.Duration(s.latency) * time.Millisecond)
		}
		if s.failRate > 0 && rand.Float64() < s.failRate {
			c.Error(500, "injected failure (FAIL_RATE)")
		} else {
			fn(c)
		}
		d := time.Since(start).Seconds()
		s.mu.Lock()
		s.reqs[fmt.Sprintf("%s|%d", r.Method, c.code)]++
		s.durSum += d
		s.durCount++
		s.mu.Unlock()
		b, _ := json.Marshal(map[string]any{"ts": time.Now().UTC().Format(time.RFC3339), "service": s.Name,
			"method": r.Method, "path": r.URL.Path, "status": c.code, "ms": int(d * 1000),
			"requestId": r.Header.Get("X-Request-Id")})
		log.Println(string(b))
	})
}

func (s *Service) metrics(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.reqs))
	for k := range s.reqs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Fprintln(w, "# TYPE railshack_http_requests_total counter")
	for _, k := range keys {
		p := strings.Split(k, "|")
		fmt.Fprintf(w, "railshack_http_requests_total{service=%q,method=%q,code=%q} %d\n", s.Name, p[0], p[1], s.reqs[k])
	}
	fmt.Fprintln(w, "# TYPE railshack_http_request_duration_seconds summary")
	fmt.Fprintf(w, "railshack_http_request_duration_seconds_sum{service=%q} %f\n", s.Name, s.durSum)
	fmt.Fprintf(w, "railshack_http_request_duration_seconds_count{service=%q} %d\n", s.Name, s.durCount)
	fmt.Fprintf(w, "railshack_up{service=%q,language=\"go\"} 1\n", s.Name)
	fmt.Fprintf(w, "railshack_build_info{service=%q,version=%q,project=%q,source=%q} 1\n", s.Name, s.Version, Brand, SourceURL)
}

// Run starts the server and shuts down gracefully on SIGTERM (what Kubernetes sends).
func (s *Service) Run() {
	log.SetFlags(0)
	port := getenv("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: s.mux}
	go func() {
		log.Printf(`{"service":%q,"msg":"listening","port":%q,"language":"go","project":%q,"youtube":%q}`, s.Name, port, Brand, YouTube)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// ---- request helpers ----

func (c *Ctx) Query(k string) string { return c.R.URL.Query().Get(k) }

// Token returns the caller's login token from "Authorization: Bearer <token>".
func (c *Ctx) Token() string {
	h := c.R.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
func (c *Ctx) Param(k string) string { return c.R.PathValue(k) }

func (c *Ctx) Bind(v any) error {
	defer c.R.Body.Close()
	return json.NewDecoder(c.R.Body).Decode(v)
}

// Work burns roughly WORK_FACTOR milliseconds of CPU. It simulates real
// business logic so CPU-based autoscaling has something to react to.
func (c *Ctx) Work() {
	if c.S.work <= 0 {
		return
	}
	buf := []byte(c.S.Name)
	for i := 0; i < c.S.work*3000; i++ {
		h := sha256.Sum256(buf)
		buf = h[:]
	}
}

func (c *Ctx) JSON(code int, v any) {
	c.code = code
	t := append([]string{c.S.Name}, c.trace...)
	c.W.Header().Set("X-Trace", strings.Join(dedupe(t), ","))
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(code)
	json.NewEncoder(c.W).Encode(v)
}

func (c *Ctx) Error(code int, msg string) {
	c.JSON(code, map[string]any{"error": msg, "service": c.S.Name})
}

// Call invokes another microservice and records it in the trace.
func (c *Ctx) Call(svc, method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(c.R.Context(), method, c.S.URL(svc)+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if id := c.R.Header.Get("X-Request-Id"); id != "" {
		req.Header.Set("X-Request-Id", id)
	}
	resp, err := c.S.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s unreachable: %w", svc, err)
	}
	defer resp.Body.Close()
	if t := resp.Header.Get("X-Trace"); t != "" {
		c.trace = append(c.trace, strings.Split(t, ",")...)
	} else {
		c.trace = append(c.trace, svc)
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s returned %d: %s", svc, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Fire calls a service but never fails the request (used for audit, analytics, notifications).
func (c *Ctx) Fire(svc, method, path string, body any) {
	if err := c.Call(svc, method, path, body, nil); err != nil {
		log.Printf(`{"service":%q,"warn":%q}`, c.S.Name, err.Error())
	}
}

// AddTrace merges a trace from a proxied response (used by the gateway).
func (c *Ctx) AddTrace(t string) {
	if t != "" {
		c.trace = append(c.trace, strings.Split(t, ",")...)
	}
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// Sub returns a child context for concurrent downstream calls; merge it back with Merge.
func (c *Ctx) Sub() *Ctx { return &Ctx{W: c.W, R: c.R, S: c.S, code: 200} }

// Merge folds a child context's trace into this one.
func (c *Ctx) Merge(subs ...*Ctx) {
	for _, s := range subs {
		c.trace = append(c.trace, s.trace...)
	}
}
