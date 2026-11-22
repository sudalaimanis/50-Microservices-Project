// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// config — central feature flags and runtime settings, overridable per environment with env vars.
package main

import (
	"os"

	"railshack/gokit"
)

func flag(k string, d bool) bool {
	if v := os.Getenv(k); v != "" {
		return v == "true"
	}
	return d
}

func main() {
	s := kit.New("config")
	s.Handle("GET /api/config/flags", func(c *kit.Ctx) {
		c.JSON(200, map[string]any{
			"bookingEnabled": flag("FLAG_BOOKING_ENABLED", true),
			"tatkalEnabled":  flag("FLAG_TATKAL_ENABLED", true),
			"foodOrdering":   flag("FLAG_FOOD_ORDERING", true),
			"saleBanner":     os.Getenv("SALE_BANNER"),
			"environment":    os.Getenv("APP_ENV"),
			"cluster":        os.Getenv("CLUSTER_NAME"),
		})
	})
	s.Run()
}
