// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// pnr — creates 10-digit PNRs and stores bookings.
//
// PostgreSQL (table bookings) is the source of truth: every booking is written there first.
// Redis is only a speed layer in front of it (cache-aside):
//
//	read   → try Redis; on a miss read PostgreSQL and keep a copy in Redis for 10 minutes
//	write  → change PostgreSQL first, then delete the Redis copy so nobody reads an old status
//
// If Redis is wiped, nothing is lost: the next read simply comes from PostgreSQL again.
// Without DATABASE_URL (local runs) bookings are kept in memory.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"railshack/gokit"
)

const cacheTTL = 600 // seconds a PNR stays in the Redis cache

type store struct {
	db     *kit.PG
	cache  *kit.Redis
	mu     sync.Mutex
	mem    map[string]string
	schema bool
}

func (s *store) ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.schema {
		return nil
	}
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS bookings (
		pnr       TEXT PRIMARY KEY,
		username  TEXT NOT NULL,
		state     TEXT NOT NULL,
		data      JSONB NOT NULL,
		booked_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	if err == nil {
		_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS bookings_username ON bookings (username)`)
	}
	s.schema = err == nil
	return err
}

func (s *store) create(pnr, user, data string) error {
	if s.db == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.mem[pnr] = data
		return nil
	}
	if err := s.ensure(); err != nil {
		return err
	}
	if _, err := s.db.Exec(`INSERT INTO bookings (pnr, username, state, data) VALUES ($1, $2, 'ACTIVE', $3)`, pnr, user, data); err != nil {
		return err
	}
	s.cache.CacheSet("pnr:"+pnr, data, cacheTTL) // warm the cache: PNR status is usually checked right after booking
	return nil
}

// get returns the booking JSON and where it came from: "redis", "postgres" or "memory".
func (s *store) get(pnr string) (string, string, error) {
	if s.db == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.mem[pnr], "memory", nil
	}
	if v, ok := s.cache.CacheGet("pnr:" + pnr); ok {
		return v, "redis", nil // cache hit: PostgreSQL is not touched
	}
	if err := s.ensure(); err != nil {
		return "", "", err
	}
	rows, err := s.db.Query(`SELECT data FROM bookings WHERE pnr = $1`, pnr)
	if err != nil || len(rows) == 0 {
		return "", "postgres", err
	}
	s.cache.CacheSet("pnr:"+pnr, rows[0]["data"], cacheTTL) // cache miss: keep a copy for next time
	return rows[0]["data"], "postgres", nil
}

func (s *store) cancel(pnr string) (string, error) {
	if s.db == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		var b map[string]any
		if json.Unmarshal([]byte(s.mem[pnr]), &b) != nil {
			return "", nil
		}
		b["state"] = "CANCELLED"
		nb, _ := json.Marshal(b)
		s.mem[pnr] = string(nb)
		return string(nb), nil
	}
	if err := s.ensure(); err != nil {
		return "", err
	}
	rows, err := s.db.Query(`UPDATE bookings SET state = 'CANCELLED', data = jsonb_set(data, '{state}', '"CANCELLED"')
		WHERE pnr = $1 RETURNING data`, pnr)
	if err != nil || len(rows) == 0 {
		return "", err
	}
	s.cache.CacheDel("pnr:" + pnr) // the database changed: throw away the old copy
	return rows[0]["data"], nil
}

func main() {
	s := kit.New("pnr")
	pg, err := kit.NewPG(os.Getenv("DATABASE_URL"), 8)
	if err != nil {
		log.Fatalf("DATABASE_URL: %v", err)
	}
	db := &store{db: pg, cache: kit.NewRedis(os.Getenv("REDIS_ADDR")), mem: map[string]string{}}

	s.Handle("POST /api/pnr/create", func(c *kit.Ctx) {
		var booking map[string]any
		if c.Bind(&booking) != nil {
			c.Error(400, "booking body required")
			return
		}
		pnr := fmt.Sprintf("%d%09d", 2+rand.Intn(7), rand.Intn(1000000000))
		booking["pnr"] = pnr
		booking["bookedAt"] = time.Now().UTC().Format(time.RFC3339)
		booking["state"] = "ACTIVE"
		b, _ := json.Marshal(booking)
		if err := db.create(pnr, fmt.Sprint(booking["user"]), string(b)); err != nil {
			c.Error(503, "booking store unavailable: "+err.Error())
			return
		}
		c.JSON(201, map[string]any{"pnr": pnr})
	})

	s.Handle("GET /api/pnr/{pnr}", func(c *kit.Ctx) {
		raw, source, err := db.get(c.Param("pnr"))
		if err != nil {
			c.Error(503, err.Error())
			return
		}
		if raw == "" {
			c.Error(404, "PNR not found")
			return
		}
		var b map[string]any
		json.Unmarshal([]byte(raw), &b)
		b["servedFrom"] = source // "redis" (cache hit) or "postgres" (cache miss)
		var chart map[string]any
		if err := c.Call("chart-preparation", "GET", fmt.Sprintf("/api/chart-preparation/status?train=%v&date=%v", b["train"], b["date"]), nil, &chart); err == nil {
			b["chart"] = chart
		}
		c.JSON(200, b)
	})

	s.Handle("POST /api/pnr/{pnr}/cancel", func(c *kit.Ctx) {
		raw, err := db.cancel(c.Param("pnr"))
		if err != nil {
			c.Error(503, err.Error())
			return
		}
		if raw == "" {
			c.Error(404, "PNR not found")
			return
		}
		var b map[string]any
		json.Unmarshal([]byte(raw), &b)
		c.JSON(200, b)
	})
	s.Run()
}
