// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// quota — reservation quota rules (General, Tatkal, Ladies, Senior Citizen).
package main

import "railshack/gokit"

var quotas = map[string]map[string]any{
	"GN": {"code": "GN", "name": "General", "share": 0.70, "openDaysBefore": 60},
	"TQ": {"code": "TQ", "name": "Tatkal", "share": 0.20, "openDaysBefore": 1, "opensAt": map[string]string{"AC": "10:00", "NON_AC": "11:00"}},
	"LD": {"code": "LD", "name": "Ladies", "share": 0.04, "openDaysBefore": 60},
	"SS": {"code": "SS", "name": "Senior Citizen", "share": 0.06, "openDaysBefore": 60},
}

func main() {
	s := kit.New("quota")
	s.Handle("GET /api/quota/rules", func(c *kit.Ctx) {
		q := c.Query("quota")
		if q == "" {
			c.JSON(200, quotas)
			return
		}
		r, ok := quotas[q]
		if !ok {
			c.Error(404, "unknown quota")
			return
		}
		c.JSON(200, r)
	})
	s.Run()
}
