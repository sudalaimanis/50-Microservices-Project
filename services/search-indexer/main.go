// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// search-indexer — builds a cached index of stations and answers type-ahead suggestions.
package main

import (
	"sort"
	"strings"
	"sync"
	"time"

	"railshack/gokit"
)

func main() {
	s := kit.New("search-indexer")
	var mu sync.Mutex
	var index []map[string]any
	var built time.Time
	s.Handle("GET /api/search-indexer/suggest", func(c *kit.Ctx) {
		mu.Lock()
		if time.Since(built) > time.Minute || index == nil {
			var st []map[string]any
			if err := c.Call("stations", "GET", "/api/stations", nil, &st); err != nil {
				mu.Unlock()
				c.Error(502, err.Error())
				return
			}
			index, built = st, time.Now()
		}
		idx := index
		mu.Unlock()
		q := strings.ToLower(strings.TrimSpace(c.Query("q")))
		type hit struct {
			s     map[string]any
			score int
		}
		hits := []hit{}
		for _, st := range idx {
			code := strings.ToLower(st["code"].(string))
			name := strings.ToLower(st["name"].(string) + " " + st["city"].(string))
			switch {
			case q == "":
				hits = append(hits, hit{st, 1})
			case code == q:
				hits = append(hits, hit{st, 100})
			case strings.HasPrefix(code, q) || strings.HasPrefix(name, q):
				hits = append(hits, hit{st, 50})
			case strings.Contains(name, q):
				hits = append(hits, hit{st, 10})
			}
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
		out := []map[string]any{}
		for i, h := range hits {
			if i == 8 {
				break
			}
			out = append(out, h.s)
		}
		c.JSON(200, out)
	})
	s.Run()
}
