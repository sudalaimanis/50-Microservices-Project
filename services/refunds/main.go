// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// refunds — returns money to the original payment method; wallet refunds are instant.
package main

import (
	"fmt"
	"time"

	"railshack/gokit"
)

func main() {
	s := kit.New("refunds")
	s.Handle("POST /api/refunds/initiate", func(c *kit.Ctx) {
		var in struct {
			User   string  `json:"user"`
			Txn    string  `json:"txn"`
			Amount float64 `json:"amount"`
			Method string  `json:"method"`
		}
		if c.Bind(&in) != nil || in.Amount < 0 {
			c.Error(400, "txn and amount are required")
			return
		}
		eta := "instant"
		var err error
		if in.Method == "wallet" {
			err = c.Call("wallet", "POST", "/api/wallet/credit", map[string]any{"user": in.User, "amount": in.Amount}, nil)
		} else {
			eta = "5-7 working days"
			err = c.Call("payment-gateway", "POST", "/api/payment-gateway/refund", map[string]any{"txn": in.Txn, "amount": in.Amount, "method": in.Method}, nil)
		}
		if err != nil {
			c.Error(502, err.Error())
			return
		}
		c.JSON(200, map[string]any{"refundId": fmt.Sprintf("RF%d", time.Now().UnixMilli()), "amount": in.Amount, "eta": eta, "status": "INITIATED"})
	})
	s.Run()
}
