// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// search — the heart of train search. Fans out to routes, trains, schedules,
// seat-map, fares and stations, then merges everything into result cards.
package main

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"railshack/gokit"
)

type Stop struct {
	Code string `json:"code"`
	Arr  string `json:"arr"`
	Dep  string `json:"dep"`
	Day  int    `json:"day"`
	Km   int    `json:"km"`
}
type Leg struct {
	Train   string `json:"train"`
	From    Stop   `json:"from"`
	To      Stop   `json:"to"`
	Km      int    `json:"km"`
	DayDiff int    `json:"dayDiff"`
}
type Train struct {
	Number  string   `json:"number"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Classes []string `json:"classes"`
	Pantry  bool     `json:"pantry"`
}

func mins(t string) int {
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return h*60 + m
}

func main() {
	s := kit.New("search")
	s.Handle("GET /api/search/trains", func(c *kit.Ctx) {
		c.Work()
		from, to, date := strings.ToUpper(c.Query("from")), strings.ToUpper(c.Query("to")), c.Query("date")
		if from == "" || to == "" || date == "" {
			c.Error(400, "from, to and date are required")
			return
		}
		var legs []Leg
		if err := c.Call("routes", "GET", "/api/routes/between?from="+from+"&to="+to, nil, &legs); err != nil {
			c.Error(502, err.Error())
			return
		}
		if len(legs) == 0 {
			c.JSON(200, map[string]any{"from": from, "to": to, "date": date, "results": []any{}})
			return
		}
		nums := []string{}
		for _, l := range legs {
			nums = append(nums, l.Train)
		}
		var (
			trains   []Train
			runs     map[string]any
			stations []map[string]any
			wg       sync.WaitGroup
			mu       sync.Mutex
			errs     []string
		)
		run := func(f func() error) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := f(); err != nil {
					mu.Lock()
					errs = append(errs, err.Error())
					mu.Unlock()
				}
			}()
		}
		// Each concurrent downstream call gets its own child context so traces merge safely.
		ct, cr, cs := c.Sub(), c.Sub(), c.Sub()
		run(func() error { return ct.Call("trains", "GET", "/api/trains?numbers="+strings.Join(nums, ","), nil, &trains) })
		run(func() error {
			return cr.Call("schedules", "GET", "/api/schedules/runs?numbers="+strings.Join(nums, ",")+"&date="+date, nil, &runs)
		})
		run(func() error { return cs.Call("stations", "GET", "/api/stations?codes="+from+","+to, nil, &stations) })
		wg.Wait()
		c.Merge(ct, cr, cs)
		if len(errs) > 0 {
			c.Error(502, strings.Join(errs, "; "))
			return
		}
		tByNo := map[string]Train{}
		items := []string{}
		fareItems := []map[string]any{}
		for _, t := range trains {
			tByNo[t.Number] = t
			items = append(items, t.Number+":"+strings.Join(t.Classes, ","))
		}
		for _, l := range legs {
			t := tByNo[l.Train]
			fareItems = append(fareItems, map[string]any{"train": l.Train, "type": t.Type, "km": l.Km, "classes": t.Classes})
		}
		var avail map[string][]map[string]any
		var fares map[string]any
		if err := c.Call("seat-map", "GET", "/api/seat-map/availability?date="+date+"&items="+url.QueryEscape(strings.Join(items, "|")), nil, &avail); err != nil {
			c.Error(502, err.Error())
			return
		}
		if err := c.Call("fares", "POST", "/api/fares/batch", map[string]any{"items": fareItems}, &fares); err != nil {
			c.Error(502, err.Error())
			return
		}
		names := map[string]string{}
		for _, st := range stations {
			names[fmt.Sprint(st["code"])] = fmt.Sprint(st["name"])
		}
		results := []map[string]any{}
		for _, l := range legs {
			t := tByNo[l.Train]
			running := true
			if r, ok := runs[l.Train].(map[string]any); ok {
				running, _ = r["runs"].(bool)
			}
			dur := mins(l.To.Arr) + (l.DayDiff * 1440) - mins(l.From.Dep)
			classes := []map[string]any{}
			fm, _ := fares[l.Train].(map[string]any)
			for _, a := range avail[l.Train] {
				cl := fmt.Sprint(a["class"])
				a["fare"] = fm[cl]
				classes = append(classes, a)
			}
			results = append(results, map[string]any{
				"train": t.Number, "name": t.Name, "type": t.Type, "pantry": t.Pantry, "runsOnDate": running,
				"from": l.From.Code, "fromName": names[l.From.Code], "to": l.To.Code, "toName": names[l.To.Code],
				"departs": l.From.Dep, "arrives": l.To.Arr, "dayDiff": l.DayDiff, "km": l.Km,
				"duration": fmt.Sprintf("%dh %02dm", dur/60, dur%60), "durationMins": dur, "classes": classes,
			})
		}
		sort.Slice(results, func(i, j int) bool { return fmt.Sprint(results[i]["departs"]) < fmt.Sprint(results[j]["departs"]) })
		c.JSON(200, map[string]any{"from": from, "fromName": names[from], "to": to, "toName": names[to], "date": date,
			"count": len(results), "results": results, "servedBy": kit.Pod()})
	})
	s.Run()
}
