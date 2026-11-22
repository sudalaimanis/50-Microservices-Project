// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// booking — the booking orchestrator. One "Book now" click flows through about twenty
// microservices: auth → profile → passengers → kyc → tatkal → fares → offers → insurance →
// seat-allocation → payment → pnr → invoices → ticket → history → notifications → audit.
package main

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"railshack/gokit"
)

type Req struct {
	Token      string           `json:"token"`
	Train      string           `json:"train"`
	From       string           `json:"from"`
	To         string           `json:"to"`
	Date       string           `json:"date"`
	Class      string           `json:"class"`
	Quota      string           `json:"quota"`
	Passengers []map[string]any `json:"passengers"`
	Payment    string           `json:"payment"`
	Coupon     string           `json:"coupon"`
	Insurance  bool             `json:"insurance"`
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}

func main() {
	s := kit.New("booking")
	s.Handle("POST /api/booking/book", func(c *kit.Ctx) {
		c.Work()
		var in Req
		if c.Bind(&in) != nil || in.Train == "" || in.Class == "" || len(in.Passengers) == 0 {
			c.Error(400, "train, class, date and at least one passenger are required")
			return
		}
		if in.Quota == "" {
			in.Quota = "GN"
		}
		if in.Payment == "" {
			in.Payment = "upi"
		}
		fail := func(step string, err error) { c.Error(502, step+": "+err.Error()) }

		// 1. Who is booking?
		var who struct {
			User string `json:"user"`
		}
		tok := c.Token() // "Authorization: Bearer <token>" (the older "token" body field still works)
		if tok == "" {
			tok = in.Token
		}
		if tok == "" {
			c.Error(401, "please sign in to book a ticket")
			return
		}
		if err := c.Call("auth", "GET", "/api/auth/verify?token="+url.QueryEscape(tok), nil, &who); err != nil {
			c.Error(401, "your session has expired, please sign in again")
			return
		}
		var profile map[string]any
		if err := c.Call("user-profile", "GET", "/api/user-profile/"+who.User, nil, &profile); err != nil {
			fail("profile", err)
			return
		}
		// 2. Validate passengers and identity.
		var pv struct {
			Passengers []map[string]any `json:"passengers"`
		}
		if err := c.Call("passengers", "POST", "/api/passengers/validate", map[string]any{"passengers": in.Passengers}, &pv); err != nil {
			c.Error(400, err.Error())
			return
		}
		var kyc map[string]any
		if err := c.Call("kyc", "GET", "/api/kyc/status/"+who.User, nil, &kyc); err != nil || kyc["verified"] != true {
			c.Error(403, "KYC verification required")
			return
		}
		// 3. Tatkal window, if Tatkal quota.
		if in.Quota == "TQ" {
			var tk map[string]any
			if err := c.Call("tatkal", "GET", "/api/tatkal/check?date="+in.Date+"&class="+in.Class, nil, &tk); err != nil {
				fail("tatkal", err)
				return
			}
			if tk["open"] != true {
				c.Error(409, fmt.Sprint(tk["message"]))
				return
			}
		}
		// 4. Price it: fare → offers → insurance.
		var route []map[string]any
		if err := c.Call("routes", "GET", "/api/routes/"+in.Train, nil, &route); err != nil {
			fail("routes", err)
			return
		}
		km := 0.0
		for _, st := range route {
			if st["code"] == in.From {
				km -= num(st["km"])
			}
			if st["code"] == in.To {
				km += num(st["km"])
			}
		}
		var train map[string]any
		c.Call("trains", "GET", "/api/trains/"+in.Train, nil, &train)
		var fare map[string]any
		if err := c.Call("fares", "POST", "/api/fares/quote", map[string]any{"type": train["type"], "km": km, "class": in.Class,
			"passengers": len(pv.Passengers), "quota": in.Quota}, &fare); err != nil {
			fail("fares", err)
			return
		}
		total := num(fare["total"])
		var offer map[string]any
		if err := c.Call("offers", "POST", "/api/offers/apply", map[string]any{"coupon": in.Coupon, "amount": total, "payment": in.Payment}, &offer); err == nil {
			total -= num(offer["discount"])
		}
		var ins map[string]any
		if in.Insurance {
			if err := c.Call("insurance", "GET", fmt.Sprintf("/api/insurance/quote?passengers=%d", len(pv.Passengers)), nil, &ins); err == nil {
				total += num(ins["premium"])
			}
		}
		total = math.Round(total*100) / 100
		// 5. Allocate seats (may fall back to RAC or waitlist).
		var alloc map[string]any
		if err := c.Call("seat-allocation", "POST", "/api/seat-allocation/allocate", map[string]any{"train": in.Train, "date": in.Date,
			"class": in.Class, "quota": in.Quota, "passengers": pv.Passengers}, &alloc); err != nil {
			fail("seat allocation", err)
			return
		}
		// 6. Take the money.
		var pay map[string]any
		if err := c.Call("payment-gateway", "POST", "/api/payment-gateway/charge", map[string]any{"user": who.User, "amount": total, "method": in.Payment}, &pay); err != nil {
			c.Error(402, err.Error())
			return
		}
		// 7. Create the PNR, invoice and ticket.
		booking := map[string]any{"user": who.User, "train": in.Train, "trainName": train["name"], "from": in.From, "to": in.To,
			"date": in.Date, "class": in.Class, "quota": in.Quota, "passengers": alloc["passengers"], "status": alloc["overall"],
			"fare": fare, "offer": offer, "insurance": ins, "total": total, "payment": pay}
		var p struct {
			PNR string `json:"pnr"`
		}
		if err := c.Call("pnr", "POST", "/api/pnr/create", booking, &p); err != nil {
			fail("pnr", err)
			return
		}
		booking["pnr"] = p.PNR
		var inv, ticket map[string]any
		c.Call("invoices", "POST", "/api/invoices/create", map[string]any{"pnr": p.PNR, "fare": fare, "total": total, "user": who.User}, &inv)
		c.Call("ticket-pdf", "POST", "/api/ticket-pdf/render", booking, &ticket)
		booking["invoice"], booking["ticket"] = inv, ticket
		// 8. Side effects that must never fail the booking.
		c.Fire("booking-history", "POST", "/api/booking-history/add", map[string]any{"user": who.User, "pnr": p.PNR, "train": in.Train,
			"trainName": train["name"], "from": in.From, "to": in.To, "date": in.Date, "class": in.Class, "status": alloc["overall"], "total": total})
		c.Fire("notifications", "POST", "/api/notifications/send", map[string]any{"user": who.User, "template": "booked", "data": map[string]any{"pnr": p.PNR}})
		c.Fire("scheduler", "POST", "/api/scheduler/jobs", map[string]any{"type": "chart-preparation", "train": in.Train, "date": in.Date})
		c.Fire("audit-log", "POST", "/api/audit-log/record", map[string]any{"actor": who.User, "action": "BOOK", "resource": "pnr/" + p.PNR})
		c.Fire("analytics", "POST", "/api/analytics/event", map[string]any{"type": "booking", "train": in.Train, "class": in.Class,
			"amount": total, "route": strings.Join([]string{in.From, in.To}, "-")})
		c.JSON(201, booking)
	})
	s.Run()
}
