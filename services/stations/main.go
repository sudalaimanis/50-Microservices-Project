// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// stations — master data for every railway station (code, name, city, zone, platforms).
package main

import (
	"sort"
	"strings"

	"railshack/gokit"
)

type Station struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	City      string `json:"city"`
	State     string `json:"state"`
	Zone      string `json:"zone"`
	Platforms int    `json:"platforms"`
}

var stations = map[string]Station{
	"NDLS": {"NDLS", "New Delhi", "New Delhi", "Delhi", "NR", 16},
	"BCT":  {"BCT", "Mumbai Central", "Mumbai", "Maharashtra", "WR", 9},
	"HWH":  {"HWH", "Howrah Junction", "Kolkata", "West Bengal", "ER", 23},
	"MAS":  {"MAS", "Chennai Central", "Chennai", "Tamil Nadu", "SR", 17},
	"SBC":  {"SBC", "KSR Bengaluru", "Bengaluru", "Karnataka", "SWR", 10},
	"BSB":  {"BSB", "Varanasi Junction", "Varanasi", "Uttar Pradesh", "NR", 9},
	"LKO":  {"LKO", "Lucknow Charbagh", "Lucknow", "Uttar Pradesh", "NR", 9},
	"CNB":  {"CNB", "Kanpur Central", "Kanpur", "Uttar Pradesh", "NCR", 10},
	"PNBE": {"PNBE", "Patna Junction", "Patna", "Bihar", "ECR", 10},
	"JP":   {"JP", "Jaipur Junction", "Jaipur", "Rajasthan", "NWR", 8},
	"ADI":  {"ADI", "Ahmedabad Junction", "Ahmedabad", "Gujarat", "WR", 12},
	"BPL":  {"BPL", "Bhopal Junction", "Bhopal", "Madhya Pradesh", "WCR", 6},
	"SC":   {"SC", "Secunderabad Junction", "Hyderabad", "Telangana", "SCR", 10},
	"PUNE": {"PUNE", "Pune Junction", "Pune", "Maharashtra", "CR", 6},
	"KOTA": {"KOTA", "Kota Junction", "Kota", "Rajasthan", "WCR", 5},
	"AGC":  {"AGC", "Agra Cantt", "Agra", "Uttar Pradesh", "NCR", 6},
	"NGP":  {"NGP", "Nagpur Junction", "Nagpur", "Maharashtra", "CR", 8},
	"BRC":  {"BRC", "Vadodara Junction", "Vadodara", "Gujarat", "WR", 7},
	"PRYJ": {"PRYJ", "Prayagraj Junction", "Prayagraj", "Uttar Pradesh", "NCR", 10},
	"ASR":  {"ASR", "Amritsar Junction", "Amritsar", "Punjab", "NR", 7},
}

func main() {
	s := kit.New("stations")
	s.Handle("GET /api/stations", func(c *kit.Ctx) {
		out := []Station{}
		if codes := c.Query("codes"); codes != "" {
			for _, code := range strings.Split(codes, ",") {
				if st, ok := stations[strings.ToUpper(code)]; ok {
					out = append(out, st)
				}
			}
		} else {
			for _, st := range stations {
				out = append(out, st)
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		}
		c.JSON(200, out)
	})
	s.Handle("GET /api/stations/{code}", func(c *kit.Ctx) {
		st, ok := stations[strings.ToUpper(c.Param("code"))]
		if !ok {
			c.Error(404, "unknown station")
			return
		}
		c.JSON(200, st)
	})
	s.Run()
}
