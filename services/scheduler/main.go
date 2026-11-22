// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// scheduler — queues background jobs such as chart preparation before departure.
package main

import (
	"fmt"
	"sync"
	"time"

	"railshack/gokit"
)

func main() {
	s := kit.New("scheduler")
	var mu sync.Mutex
	jobs := []map[string]any{}
	s.Handle("POST /api/scheduler/jobs", func(c *kit.Ctx) {
		var j map[string]any
		if c.Bind(&j) != nil {
			c.Error(400, "job required")
			return
		}
		mu.Lock()
		j["id"] = fmt.Sprintf("JOB-%d", len(jobs)+1)
		j["queuedAt"] = time.Now().UTC().Format(time.RFC3339)
		j["runAt"] = fmt.Sprintf("%v (4 hours before departure)", j["date"])
		jobs = append(jobs, j)
		if len(jobs) > 300 {
			jobs = jobs[1:]
		}
		mu.Unlock()
		c.JSON(201, j)
	})
	s.Handle("GET /api/scheduler/jobs", func(c *kit.Ctx) {
		mu.Lock()
		defer mu.Unlock()
		c.JSON(200, map[string]any{"queued": len(jobs)})
	})
	s.Run()
}
