// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// cab-booking — last-mile cab quotes from the destination station.
package main

import (
	"fmt"
	"math"
	"strconv"

	"railshack/gokit"
)

func main() {
	s := kit.New("cab-booking")
	s.Handle("GET /api/cab-booking/quote", func(c *kit.Ctx) {
		var st map[string]any
		if err := c.Call("stations", "GET", "/api/stations/"+c.Query("station"), nil, &st); err != nil {
			c.Error(404, err.Error())
			return
		}
		km, _ := strconv.ParseFloat(c.Query("km"), 64)
		if km <= 0 {
			km = 12
		}
		opts := []map[string]any{}
		for i, t := range []struct {
			name  string
			base  float64
			perKm float64
			seats int
		}{{"Auto", 30, 11, 3}, {"Mini", 60, 14, 4}, {"Sedan", 90, 18, 4}, {"SUV", 140, 24, 6}} {
			opts = append(opts, map[string]any{"type": t.name, "seats": t.seats, "fare": math.Round(t.base + t.perKm*km),
				"etaMins": 3 + i*2, "label": fmt.Sprintf("%s · %d seats", t.name, t.seats)})
		}
		c.JSON(200, map[string]any{"pickup": st["name"], "city": st["city"], "km": km, "options": opts})
	})
	s.Run()
}
