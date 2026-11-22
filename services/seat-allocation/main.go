// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// seat-allocation — assigns coach and berth numbers. Uses seat-map for the coach layout and
// quota for the quota share; passengers it cannot confirm go to RAC, then to the waitlist.
package main

import (
	"fmt"
	"hash/fnv"
	"strings"

	"railshack/gokit"
)

func h(s string) uint32 { f := fnv.New32a(); f.Write([]byte(s)); return f.Sum32() }

func main() {
	s := kit.New("seat-allocation")
	s.Handle("POST /api/seat-allocation/allocate", func(c *kit.Ctx) {
		c.Work()
		var in struct {
			Train      string           `json:"train"`
			Date       string           `json:"date"`
			Class      string           `json:"class"`
			Quota      string           `json:"quota"`
			Passengers []map[string]any `json:"passengers"`
		}
		if c.Bind(&in) != nil || len(in.Passengers) == 0 {
			c.Error(400, "train, date, class and passengers are required")
			return
		}
		var layout struct {
			CoachPrefix    string   `json:"coachPrefix"`
			Coaches        int      `json:"coaches"`
			BerthsPerCoach int      `json:"berthsPerCoach"`
			Berths         []string `json:"berths"`
		}
		if err := c.Call("seat-map", "GET", "/api/seat-map/coach-layout?class="+in.Class, nil, &layout); err != nil {
			c.Error(502, err.Error())
			return
		}
		var avail map[string][]map[string]any
		c.Call("seat-map", "GET", "/api/seat-map/availability?date="+in.Date+"&items="+in.Train+":"+in.Class, nil, &avail)
		c.Call("quota", "GET", "/api/quota/rules?quota="+in.Quota, nil, nil)
		status, count := "AVAILABLE", 999
		if a := avail[in.Train]; len(a) > 0 {
			status = fmt.Sprint(a[0]["status"])
			fmt.Sscan(fmt.Sprint(a[0]["count"]), &count)
		}
		out := []map[string]any{}
		for i, p := range in.Passengers {
			name := fmt.Sprint(p["name"])
			if status == "AVAILABLE" && i < count {
				seed := h(fmt.Sprintf("%s%s%s%s%d", in.Train, in.Date, in.Class, name, i))
				coach := fmt.Sprintf("%s%d", layout.CoachPrefix, int(seed)%layout.Coaches+1)
				berthNo := int(seed/7)%layout.BerthsPerCoach + 1
				berth := layout.Berths[berthNo%len(layout.Berths)]
				out = append(out, map[string]any{"name": name, "status": "CNF", "coach": coach, "berth": berthNo,
					"berthType": berth, "label": fmt.Sprintf("CNF/%s/%d/%s", coach, berthNo, berth)})
				continue
			}
			var r map[string]any
			if status != "WL" {
				if err := c.Call("rac", "POST", "/api/rac/assign", map[string]any{"train": in.Train, "date": in.Date, "class": in.Class, "name": name}, &r); err == nil && r["status"] == "RAC" {
					out = append(out, r)
					continue
				}
			}
			if err := c.Call("waitlist", "POST", "/api/waitlist/add", map[string]any{"train": in.Train, "date": in.Date, "class": in.Class, "name": name}, &r); err != nil {
				c.Error(502, err.Error())
				return
			}
			out = append(out, r)
		}
		overall := "CONFIRMED"
		for _, p := range out {
			if st := fmt.Sprint(p["status"]); st != "CNF" {
				overall = map[bool]string{true: "RAC", false: "WAITLISTED"}[strings.HasPrefix(st, "RAC")]
			}
		}
		c.JSON(200, map[string]any{"passengers": out, "overall": overall})
	})
	s.Run()
}
