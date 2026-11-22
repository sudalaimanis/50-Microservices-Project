// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// api-gateway — the single front door. Applies rate limiting, routes /api/<service>/...
// to the right microservice, and exposes /api/platform/status: a live health sweep
// across all 50 services that powers the animated mesh in the UI.
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"railshack/gokit"
)

var services = []string{"auth", "user-profile", "passengers", "kyc", "notifications", "preferences",
	"stations", "trains", "routes", "schedules", "search", "seat-map", "fares", "quota",
	"booking", "pnr", "seat-allocation", "waitlist", "rac", "tatkal", "cancellation", "refunds", "booking-history", "ticket-pdf",
	"payment-gateway", "wallet", "upi", "invoices", "offers", "settlements",
	"live-status", "platform-info", "food-ordering", "retiring-rooms", "tourism", "cab-booking", "insurance", "chart-preparation",
	"api-gateway", "config", "rate-limiter", "audit-log", "analytics", "search-indexer", "email", "sms", "scheduler", "feedback", "admin", "frontend"}

func main() {
	s := kit.New("api-gateway")
	allowed := map[string]bool{}
	for _, n := range services {
		allowed[n] = true
	}
	client := &http.Client{Timeout: 10 * time.Second}
	probe := &http.Client{Timeout: 1500 * time.Millisecond}

	s.Handle("GET /api/platform/status", func(c *kit.Ctx) {
		type row struct {
			Service   string `json:"service"`
			Up        bool   `json:"up"`
			Pod       string `json:"pod,omitempty"`
			Language  string `json:"language,omitempty"`
			Version   string `json:"version,omitempty"`
			LatencyMs int64  `json:"latencyMs"`
		}
		rows := make([]row, len(services))
		var wg sync.WaitGroup
		for i, name := range services {
			wg.Add(1)
			go func(i int, name string) {
				defer wg.Done()
				start := time.Now()
				r := row{Service: name}
				if name == "api-gateway" {
					r.Up, r.Pod, r.Language, r.Version = true, kit.Pod(), "go", s.Version
				} else if resp, err := probe.Get(s.URL(name) + "/api/" + name + "/info"); err == nil {
					var info map[string]any
					json.NewDecoder(resp.Body).Decode(&info)
					resp.Body.Close()
					r.Up = resp.StatusCode == 200
					r.Pod, _ = info["pod"].(string)
					r.Language, _ = info["language"].(string)
					r.Version, _ = info["version"].(string)
				}
				r.LatencyMs = time.Since(start).Milliseconds()
				rows[i] = r
			}(i, name)
		}
		wg.Wait()
		up := 0
		for _, r := range rows {
			if r.Up {
				up++
			}
		}
		c.JSON(200, map[string]any{"total": len(rows), "up": up, "services": rows, "gatewayPod": kit.Pod(), "checkedAt": time.Now().UTC()})
	})

	s.Handle("/api/", func(c *kit.Ctx) {
		parts := strings.SplitN(strings.TrimPrefix(c.R.URL.Path, "/api/"), "/", 2)
		target := parts[0]
		if !allowed[target] || target == "api-gateway" || target == "frontend" {
			c.Error(404, "no such service: "+target)
			return
		}
		if target != "rate-limiter" {
			key := c.R.Header.Get("X-Forwarded-For")
			if key == "" {
				key = strings.Split(c.R.RemoteAddr, ":")[0]
			}
			if err := c.Call("rate-limiter", "GET", "/api/rate-limiter/check?key="+strings.Split(key, ",")[0], nil, nil); err != nil && strings.Contains(err.Error(), "429") {
				c.Error(429, "slow down — rate limit exceeded")
				return
			}
		}
		body, _ := io.ReadAll(c.R.Body)
		req, _ := http.NewRequestWithContext(c.R.Context(), c.R.Method, s.URL(target)+c.R.URL.RequestURI(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-Id", c.R.Header.Get("X-Request-Id"))
		if a := c.R.Header.Get("Authorization"); a != "" {
			req.Header.Set("Authorization", a) // services decide for themselves who may do what
		}
		resp, err := client.Do(req)
		if err != nil {
			c.Error(502, target+" unreachable: "+err.Error())
			return
		}
		defer resp.Body.Close()
		c.AddTrace(resp.Header.Get("X-Trace"))
		var out any
		raw, _ := io.ReadAll(resp.Body)
		if json.Unmarshal(raw, &out) != nil {
			out = map[string]any{"raw": string(raw)}
		}
		if p := resp.Header.Get("X-Pod"); p != "" {
			c.W.Header().Set("X-Upstream-Pod", p)
		}
		c.JSON(resp.StatusCode, out)
	})
	s.Run()
}
