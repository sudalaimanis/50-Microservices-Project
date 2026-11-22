// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// payment-gateway — one entry point for every payment method. Routes to upi or wallet,
// handles cards itself (mock), then records the transaction with settlements.
package main

import (
	"fmt"
	"math/rand"
	"time"

	"railshack/gokit"
)

func main() {
	s := kit.New("payment-gateway")
	s.Handle("POST /api/payment-gateway/charge", func(c *kit.Ctx) {
		c.Work()
		var in struct {
			User   string  `json:"user"`
			Amount float64 `json:"amount"`
			Method string  `json:"method"`
			Ref    string  `json:"ref"`
		}
		if c.Bind(&in) != nil || in.Amount <= 0 {
			c.Error(400, "user, amount and method are required")
			return
		}
		txn := fmt.Sprintf("TXN%d%04d", time.Now().UnixMilli(), rand.Intn(10000))
		var err error
		switch in.Method {
		case "upi":
			err = c.Call("upi", "POST", "/api/upi/collect", map[string]any{"user": in.User, "amount": in.Amount, "txn": txn}, nil)
		case "wallet":
			err = c.Call("wallet", "POST", "/api/wallet/debit", map[string]any{"user": in.User, "amount": in.Amount}, nil)
		case "card":
			time.Sleep(120 * time.Millisecond) // simulated 3-D Secure round trip
		default:
			c.Error(400, "method must be upi, wallet or card")
			return
		}
		if err != nil {
			c.Error(402, "payment failed: "+err.Error())
			return
		}
		c.Fire("settlements", "POST", "/api/settlements/record", map[string]any{"txn": txn, "amount": in.Amount, "method": in.Method, "type": "CHARGE"})
		c.JSON(200, map[string]any{"txn": txn, "status": "SUCCESS", "amount": in.Amount, "method": in.Method})
	})
	s.Handle("POST /api/payment-gateway/refund", func(c *kit.Ctx) {
		var in struct {
			Txn    string  `json:"txn"`
			Amount float64 `json:"amount"`
			Method string  `json:"method"`
		}
		c.Bind(&in)
		c.Fire("settlements", "POST", "/api/settlements/record", map[string]any{"txn": in.Txn, "amount": in.Amount, "method": in.Method, "type": "REFUND"})
		c.JSON(200, map[string]any{"txn": in.Txn, "status": "REFUNDED", "amount": in.Amount, "eta": "5-7 working days"})
	})
	s.Run()
}
