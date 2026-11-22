// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// wallet — RailShack wallet balances: debit for bookings, credit for refunds.
package main

import (
	"sync"

	"railshack/gokit"
)

func main() {
	s := kit.New("wallet")
	var mu sync.Mutex
	bal := map[string]float64{}
	get := func(u string) float64 {
		if _, ok := bal[u]; !ok {
			bal[u] = 25000
		}
		return bal[u]
	}
	s.Handle("GET /api/wallet/balance/{user}", func(c *kit.Ctx) {
		mu.Lock()
		defer mu.Unlock()
		c.JSON(200, map[string]any{"user": c.Param("user"), "balance": get(c.Param("user")), "currency": "INR"})
	})
	move := func(sign float64) func(*kit.Ctx) {
		return func(c *kit.Ctx) {
			var in struct {
				User   string  `json:"user"`
				Amount float64 `json:"amount"`
			}
			if c.Bind(&in) != nil || in.User == "" || in.Amount <= 0 {
				c.Error(400, "user and a positive amount are required")
				return
			}
			mu.Lock()
			defer mu.Unlock()
			b := get(in.User)
			if sign < 0 && b < in.Amount {
				c.Error(402, "insufficient wallet balance")
				return
			}
			bal[in.User] = b + sign*in.Amount
			c.JSON(200, map[string]any{"user": in.User, "balance": bal[in.User], "status": "SUCCESS"})
		}
	}
	s.Handle("POST /api/wallet/debit", move(-1))
	s.Handle("POST /api/wallet/credit", move(1))
	s.Run()
}
