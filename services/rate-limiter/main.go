// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// rate-limiter — a token bucket per client key. Returns 429 when a client goes too fast.
package main

import (
	"os"
	"strconv"
	"sync"
	"time"

	"railshack/gokit"
)

type bucket struct {
	tokens float64
	last   time.Time
}

func main() {
	s := kit.New("rate-limiter")
	rps, _ := strconv.ParseFloat(os.Getenv("RATE_LIMIT_RPS"), 64)
	if rps <= 0 {
		rps = 1000
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	s.Handle("GET /api/rate-limiter/check", func(c *kit.Ctx) {
		key := c.Query("key")
		mu.Lock()
		b, ok := buckets[key]
		now := time.Now()
		if !ok {
			b = &bucket{tokens: rps, last: now}
			buckets[key] = b
		}
		b.tokens = min(rps, b.tokens+now.Sub(b.last).Seconds()*rps)
		b.last = now
		allowed := b.tokens >= 1
		if allowed {
			b.tokens--
		}
		remaining := int(b.tokens)
		mu.Unlock()
		if !allowed {
			c.JSON(429, map[string]any{"allowed": false, "key": key})
			return
		}
		c.JSON(200, map[string]any{"allowed": true, "remaining": remaining, "limitPerSecond": rps})
	})
	s.Run()
}
