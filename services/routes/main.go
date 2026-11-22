// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// routes — the stop sequence of every train, and which trains connect two stations.
package main

import (
	"strings"

	"railshack/gokit"
)

type Stop struct {
	Code string `json:"code"`
	Arr  string `json:"arr"`
	Dep  string `json:"dep"`
	Day  int    `json:"day"`
	Km   int    `json:"km"`
}

func st(code, arr, dep string, day, km int) Stop { return Stop{code, arr, dep, day, km} }

var routes = map[string][]Stop{
	"12301": {st("HWH", "", "16:50", 0, 0), st("PNBE", "21:55", "22:05", 0, 532), st("CNB", "04:50", "04:55", 1, 1007), st("NDLS", "10:00", "", 1, 1447)},
	"12302": {st("NDLS", "", "16:55", 0, 0), st("CNB", "21:50", "21:55", 0, 440), st("PNBE", "03:40", "03:50", 1, 915), st("HWH", "09:55", "", 1, 1447)},
	"12951": {st("BCT", "", "17:00", 0, 0), st("BRC", "20:52", "20:57", 0, 392), st("KOTA", "03:15", "03:25", 1, 920), st("NDLS", "08:30", "", 1, 1386)},
	"12952": {st("NDLS", "", "16:55", 0, 0), st("KOTA", "21:35", "21:45", 0, 466), st("BRC", "03:50", "03:55", 1, 994), st("BCT", "08:35", "", 1, 1386)},
	"22436": {st("NDLS", "", "06:00", 0, 0), st("CNB", "10:08", "10:10", 0, 440), st("PRYJ", "12:08", "12:10", 0, 634), st("BSB", "14:00", "", 0, 759)},
	"22435": {st("BSB", "", "15:00", 0, 0), st("PRYJ", "16:38", "16:40", 0, 125), st("CNB", "18:58", "19:00", 0, 319), st("NDLS", "23:00", "", 0, 759)},
	"12560": {st("NDLS", "", "20:05", 0, 0), st("CNB", "02:20", "02:25", 1, 440), st("PRYJ", "04:55", "05:00", 1, 634), st("BSB", "07:55", "", 1, 759)},
	"12559": {st("BSB", "", "19:30", 0, 0), st("PRYJ", "21:50", "22:00", 0, 125), st("CNB", "01:00", "01:05", 1, 319), st("NDLS", "07:00", "", 1, 759)},
	"12002": {st("NDLS", "", "06:00", 0, 0), st("AGC", "07:50", "07:55", 0, 195), st("BPL", "14:25", "", 0, 702)},
	"12627": {st("SBC", "", "19:20", 0, 0), st("SC", "05:30", "05:45", 1, 610), st("NGP", "15:00", "15:10", 1, 1190), st("BPL", "22:10", "22:20", 1, 1580), st("AGC", "04:50", "04:55", 2, 2083), st("NDLS", "07:40", "", 2, 2278)},
	"12839": {st("HWH", "", "23:45", 0, 0), st("MAS", "04:30", "", 2, 1662)},
	"12123": {st("BCT", "", "17:10", 0, 0), st("PUNE", "20:25", "", 0, 192)},
	"12957": {st("ADI", "", "17:40", 0, 0), st("JP", "04:45", "04:55", 1, 630), st("NDLS", "09:30", "", 1, 938)},
	"12013": {st("NDLS", "", "16:30", 0, 0), st("ASR", "22:30", "", 0, 449)},
	"12015": {st("NDLS", "", "06:10", 0, 0), st("JP", "10:40", "", 0, 308)},
	"22691": {st("SBC", "", "20:00", 0, 0), st("SC", "06:45", "06:55", 1, 610), st("NGP", "15:40", "15:50", 1, 1190), st("BPL", "21:50", "22:00", 1, 1580), st("NDLS", "05:55", "", 2, 2365)},
	"12391": {st("PNBE", "", "07:30", 0, 0), st("BSB", "13:15", "13:25", 0, 228), st("LKO", "19:00", "19:10", 0, 514), st("NDLS", "05:10", "", 1, 1006)},
	"12230": {st("NDLS", "", "22:00", 0, 0), st("LKO", "06:55", "", 1, 512)},
}

type Leg struct {
	Train   string `json:"train"`
	From    Stop   `json:"from"`
	To      Stop   `json:"to"`
	Km      int    `json:"km"`
	DayDiff int    `json:"dayDiff"`
}

func main() {
	s := kit.New("routes")
	s.Handle("GET /api/routes/between", func(c *kit.Ctx) {
		from, to := strings.ToUpper(c.Query("from")), strings.ToUpper(c.Query("to"))
		legs := []Leg{}
		for no, stops := range routes {
			fi := -1
			for i, x := range stops {
				if x.Code == from {
					fi = i
				}
				if x.Code == to && fi >= 0 && i > fi {
					legs = append(legs, Leg{no, stops[fi], x, x.Km - stops[fi].Km, x.Day - stops[fi].Day})
					break
				}
			}
		}
		c.JSON(200, legs)
	})
	s.Handle("GET /api/routes/{train}", func(c *kit.Ctx) {
		r, ok := routes[c.Param("train")]
		if !ok {
			c.Error(404, "unknown train")
			return
		}
		c.JSON(200, r)
	})
	s.Run()
}
