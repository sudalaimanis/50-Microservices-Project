// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// seat-map — live seat availability per train, date and class, plus coach layouts.
// Availability is derived deterministically, so every replica gives the same answer.
package main

import (
	"fmt"
	"hash/fnv"
	"strings"

	"railshack/gokit"
)

var layouts = map[string]map[string]any{
	"1A": {"coachPrefix": "H", "coaches": 1, "berthsPerCoach": 24, "berths": []string{"LB", "UB"}},
	"2A": {"coachPrefix": "A", "coaches": 2, "berthsPerCoach": 48, "berths": []string{"LB", "UB", "SL", "SU"}},
	"3A": {"coachPrefix": "B", "coaches": 6, "berthsPerCoach": 72, "berths": []string{"LB", "MB", "UB", "SL", "SU"}},
	"SL": {"coachPrefix": "S", "coaches": 10, "berthsPerCoach": 72, "berths": []string{"LB", "MB", "UB", "SL", "SU"}},
	"CC": {"coachPrefix": "C", "coaches": 12, "berthsPerCoach": 78, "berths": []string{"W", "M", "A"}},
	"EC": {"coachPrefix": "E", "coaches": 2, "berthsPerCoach": 52, "berths": []string{"W", "A"}},
	"2S": {"coachPrefix": "D", "coaches": 8, "berthsPerCoach": 108, "berths": []string{"W", "M", "A"}},
}

func h(s string) uint32 { f := fnv.New32a(); f.Write([]byte(s)); return f.Sum32() }

func availability(train, date, class string) map[string]any {
	v := int(h(train+date+class) % 260)
	switch {
	case v < 150:
		return map[string]any{"class": class, "status": "AVAILABLE", "count": v + 4, "label": fmt.Sprintf("AVL %d", v+4)}
	case v < 190:
		return map[string]any{"class": class, "status": "RAC", "count": v - 149, "label": fmt.Sprintf("RAC %d", v-149)}
	default:
		return map[string]any{"class": class, "status": "WL", "count": v - 189, "label": fmt.Sprintf("WL %d", v-189)}
	}
}

func main() {
	s := kit.New("seat-map")
	s.Handle("GET /api/seat-map/availability", func(c *kit.Ctx) {
		c.Work()
		date := c.Query("date")
		out := map[string][]map[string]any{}
		for _, item := range strings.Split(c.Query("items"), "|") { // items=12301:1A,2A|22436:CC
			p := strings.SplitN(item, ":", 2)
			if len(p) != 2 {
				continue
			}
			for _, cl := range strings.Split(p[1], ",") {
				out[p[0]] = append(out[p[0]], availability(p[0], date, cl))
			}
		}
		c.JSON(200, out)
	})
	s.Handle("GET /api/seat-map/coach-layout", func(c *kit.Ctx) {
		l, ok := layouts[c.Query("class")]
		if !ok {
			c.Error(404, "unknown class")
			return
		}
		c.JSON(200, l)
	})
	s.Run()
}
