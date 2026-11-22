// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// live-status — simulated "where is my train" using the route, the schedule and the clock.
package main

import (
	"fmt"
	"hash/fnv"
	"time"

	"railshack/gokit"
)

type Stop struct {
	Code string `json:"code"`
	Arr  string `json:"arr"`
	Dep  string `json:"dep"`
	Day  int    `json:"day"`
	Km   int    `json:"km"`
}

func mins(t string) int {
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return h*60 + m
}

func main() {
	s := kit.New("live-status")
	ist := time.FixedZone("IST", 5*3600+1800)
	s.Handle("GET /api/live-status/{train}", func(c *kit.Ctx) {
		var stops []Stop
		if err := c.Call("routes", "GET", "/api/routes/"+c.Param("train"), nil, &stops); err != nil {
			c.Error(404, err.Error())
			return
		}
		var train map[string]any
		c.Call("trains", "GET", "/api/trains/"+c.Param("train"), nil, &train)
		f := fnv.New32a()
		f.Write([]byte(c.Param("train") + time.Now().In(ist).Format("2006-01-02")))
		delay := int(f.Sum32() % 26)
		now := time.Now().In(ist)
		nowM := now.Hour()*60 + now.Minute()
		start := mins(stops[0].Dep)
		total := mins(stops[len(stops)-1].Arr) + stops[len(stops)-1].Day*1440 - start
		// Pretend the train left on the most recent departure, so the demo always shows motion.
		elapsed := ((nowM-start)%1440 + 1440) % 1440
		if total > 1440 {
			elapsed = (elapsed + 1440*(now.YearDay()%(total/1440+1))) % (total + 60)
		}
		elapsed -= delay
		out := []map[string]any{}
		current, next := stops[0].Code, ""
		for _, st := range stops {
			t := mins(st.Arr)
			if st.Arr == "" {
				t = mins(st.Dep)
			}
			off := t + st.Day*1440 - start
			state := "upcoming"
			if off <= elapsed {
				state, current = "departed", st.Code
			} else if next == "" {
				next = st.Code
			}
			out = append(out, map[string]any{"code": st.Code, "arr": st.Arr, "dep": st.Dep, "day": st.Day, "km": st.Km, "state": state})
		}
		progress := 100.0
		if total > 0 && elapsed < total {
			progress = float64(max(elapsed, 0)) * 100 / float64(total)
		}
		var platform map[string]any
		if next != "" {
			c.Call("platform-info", "GET", "/api/platform-info/"+next+"?train="+c.Param("train"), nil, &platform)
		}
		c.JSON(200, map[string]any{"train": c.Param("train"), "name": train["name"], "delayMins": delay, "lastStation": current,
			"nextStation": next, "progress": progress, "stops": out, "platform": platform, "updatedAt": now.Format("15:04")})
	})
	s.Run()
}
