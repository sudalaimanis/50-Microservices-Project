// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// trains — train master: number, name, category and the classes each train carries.
package main

import (
	"sort"
	"strings"

	"railshack/gokit"
)

type Train struct {
	Number   string   `json:"number"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Classes  []string `json:"classes"`
	Pantry   bool     `json:"pantry"`
	MaxSpeed int      `json:"maxSpeedKmph"`
}

var trains = []Train{
	{"12301", "Howrah Rajdhani Express", "Rajdhani", []string{"1A", "2A", "3A"}, true, 130},
	{"12302", "New Delhi Rajdhani Express", "Rajdhani", []string{"1A", "2A", "3A"}, true, 130},
	{"12951", "Mumbai Rajdhani Express", "Rajdhani", []string{"1A", "2A", "3A"}, true, 130},
	{"12952", "New Delhi Mumbai Rajdhani", "Rajdhani", []string{"1A", "2A", "3A"}, true, 130},
	{"22436", "Vande Bharat Express", "Vande Bharat", []string{"CC", "EC"}, true, 160},
	{"22435", "Vande Bharat Express", "Vande Bharat", []string{"CC", "EC"}, true, 160},
	{"12560", "Shiv Ganga Express", "Superfast", []string{"SL", "3A", "2A", "1A"}, true, 110},
	{"12559", "Shiv Ganga Express", "Superfast", []string{"SL", "3A", "2A", "1A"}, true, 110},
	{"12002", "Bhopal Shatabdi Express", "Shatabdi", []string{"CC", "EC"}, true, 150},
	{"12627", "Karnataka Express", "Superfast", []string{"SL", "3A", "2A"}, true, 110},
	{"12839", "Howrah Chennai Mail", "Mail", []string{"SL", "3A", "2A"}, true, 110},
	{"12123", "Deccan Queen", "Superfast", []string{"2S", "CC"}, false, 110},
	{"12957", "Swarna Jayanti Rajdhani", "Rajdhani", []string{"1A", "2A", "3A"}, true, 130},
	{"12013", "Amritsar Shatabdi Express", "Shatabdi", []string{"CC", "EC"}, true, 150},
	{"12015", "Ajmer Shatabdi Express", "Shatabdi", []string{"CC", "EC"}, true, 150},
	{"22691", "Bengaluru Rajdhani Express", "Rajdhani", []string{"1A", "2A", "3A"}, true, 130},
	{"12391", "Shramjeevi Express", "Superfast", []string{"SL", "3A", "2A"}, true, 110},
	{"12230", "Lucknow Mail", "Mail", []string{"SL", "3A", "2A", "1A"}, false, 110},
}

func main() {
	s := kit.New("trains")
	byNo := map[string]Train{}
	for _, t := range trains {
		byNo[t.Number] = t
	}
	s.Handle("GET /api/trains", func(c *kit.Ctx) {
		out := []Train{}
		if nums := c.Query("numbers"); nums != "" {
			for _, n := range strings.Split(nums, ",") {
				if t, ok := byNo[n]; ok {
					out = append(out, t)
				}
			}
		} else {
			out = append(out, trains...)
			sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
		}
		c.JSON(200, out)
	})
	s.Handle("GET /api/trains/{number}", func(c *kit.Ctx) {
		t, ok := byNo[c.Param("number")]
		if !ok {
			c.Error(404, "unknown train")
			return
		}
		c.JSON(200, t)
	})
	s.Run()
}
