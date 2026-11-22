// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// notifications — decides which channels to use (from preferences) and fans out to email and sms.
package main

import (
	"fmt"

	"railshack/gokit"
)

func main() {
	s := kit.New("notifications")
	s.Handle("POST /api/notifications/send", func(c *kit.Ctx) {
		var in struct {
			User     string         `json:"user"`
			Template string         `json:"template"`
			Data     map[string]any `json:"data"`
		}
		if c.Bind(&in) != nil || in.User == "" {
			c.Error(400, "user and template are required")
			return
		}
		var prefs struct {
			Channels []string `json:"channels"`
		}
		if err := c.Call("preferences", "GET", "/api/preferences/"+in.User, nil, &prefs); err != nil {
			prefs.Channels = []string{"email"}
		}
		subject := map[string]string{"booked": "Your ticket is confirmed", "cancelled": "Your ticket is cancelled"}[in.Template]
		if subject == "" {
			subject = "RailShack update"
		}
		sent := []string{}
		for _, ch := range prefs.Channels {
			msg := map[string]any{"to": in.User, "subject": subject, "body": fmt.Sprintf("%s — PNR %v", subject, in.Data["pnr"])}
			if ch == "email" || ch == "sms" {
				if err := c.Call(ch, "POST", "/api/"+ch+"/send", msg, nil); err == nil {
					sent = append(sent, ch)
				}
			}
		}
		c.JSON(200, map[string]any{"sent": sent, "template": in.Template})
	})
	s.Run()
}
