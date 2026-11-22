// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// audit-log — append-only trail of who did what (bookings, cancellations, admin actions).
package main

import (
	"sync"
	"time"

	"railshack/gokit"
)

func main() {
	s := kit.New("audit-log")
	var mu sync.Mutex
	entries := []map[string]any{}
	s.Handle("POST /api/audit-log/record", func(c *kit.Ctx) {
		var e map[string]any
		if c.Bind(&e) != nil {
			c.Error(400, "entry required")
			return
		}
		e["at"] = time.Now().UTC().Format(time.RFC3339)
		mu.Lock()
		entries = append(entries, e)
		if len(entries) > 500 {
			entries = entries[len(entries)-500:]
		}
		mu.Unlock()
		c.JSON(201, map[string]any{"recorded": true})
	})
	s.Handle("GET /api/audit-log/recent", func(c *kit.Ctx) {
		mu.Lock()
		defer mu.Unlock()
		n := len(entries)
		out := []map[string]any{}
		for i := n - 1; i >= 0 && i >= n-20; i-- {
			out = append(out, entries[i])
		}
		c.JSON(200, map[string]any{"total": n, "recent": out})
	})
	s.Run()
}
