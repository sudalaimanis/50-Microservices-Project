// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// auth — sign up and sign in with a username and password. Accounts live in PostgreSQL (table users),
// the source of truth every replica shares; passwords are stored only as a salted PBKDF2-SHA256 hash. Tokens are
// stateless and HMAC-signed, so any replica can verify them without a database lookup.
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"railshack/gokit"
)

const iterations = 60000

var secret = []byte(func() string {
	if v := os.Getenv("AUTH_SECRET"); v != "" {
		return v
	}
	return "railshack-demo-secret" // Helm sets AUTH_SECRET from a Kubernetes Secret
}())

var validUser = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,23}$`)

type account struct {
	User      string `json:"user"`
	Name      string `json:"name"`
	Salt      string `json:"salt"`
	Hash      string `json:"hash"`
	Iter      int    `json:"iter"`
	CreatedAt string `json:"createdAt"`
}

// ---------- users store: PostgreSQL when DATABASE_URL is set, memory otherwise ----------

type store struct {
	db     *kit.PG
	mu     sync.Mutex
	mem    map[string]string
	schema bool
}

// ensure creates the users table the first time the database is reachable.
func (s *store) ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.schema {
		return nil
	}
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS users (
		username   TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		salt       TEXT NOT NULL,
		hash       TEXT NOT NULL,
		iterations INT  NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	s.schema = err == nil
	return err
}

// create stores the account only if the username is free. It reports false if taken.
func (s *store) create(a account) (bool, error) {
	if s.db != nil {
		if err := s.ensure(); err != nil {
			return false, err
		}
		n, err := s.db.Exec(`INSERT INTO users (username, name, salt, hash, iterations) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (username) DO NOTHING`, a.User, a.Name, a.Salt, a.Hash, a.Iter)
		return n == 1, err
	}
	b, _ := json.Marshal(a)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mem[a.User]; ok {
		return false, nil
	}
	s.mem[a.User] = string(b)
	return true, nil
}

func (s *store) get(user string) (*account, error) {
	if s.db != nil {
		if err := s.ensure(); err != nil {
			return nil, err
		}
		rows, err := s.db.Query(`SELECT username, name, salt, hash, iterations FROM users WHERE username = $1`, user)
		if err != nil || len(rows) == 0 {
			return nil, err
		}
		r := rows[0]
		iter, _ := strconv.Atoi(r["iterations"])
		return &account{User: r["username"], Name: r["name"], Salt: r["salt"], Hash: r["hash"], Iter: iter}, nil
	}
	s.mu.Lock()
	raw := s.mem[user]
	s.mu.Unlock()
	if raw == "" {
		return nil, nil
	}
	var a account
	return &a, json.Unmarshal([]byte(raw), &a)
}

// ---------- passwords and tokens ----------

// pbkdf2 is PBKDF2-HMAC-SHA256 with one 32-byte block (standard library only).
func pbkdf2(password, salt []byte, iter int) []byte {
	m := hmac.New(sha256.New, password)
	m.Write(salt)
	m.Write([]byte{0, 0, 0, 1})
	u := m.Sum(nil)
	out := append([]byte(nil), u...)
	for i := 1; i < iter; i++ {
		m.Reset()
		m.Write(u)
		u = m.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return out
}

func newAccount(user, name, password string) account {
	salt := make([]byte, 16)
	rand.Read(salt)
	return account{User: user, Name: name, Salt: base64.StdEncoding.EncodeToString(salt),
		Hash: base64.StdEncoding.EncodeToString(pbkdf2([]byte(password), salt, iterations)), Iter: iterations,
		CreatedAt: time.Now().UTC().Format(time.RFC3339)}
}

func (a *account) check(password string) bool {
	salt, _ := base64.StdEncoding.DecodeString(a.Salt)
	want, _ := base64.StdEncoding.DecodeString(a.Hash)
	return hmac.Equal(pbkdf2([]byte(password), salt, a.Iter), want)
}

func sign(payload string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))[:22]
}

func issue(c *kit.Ctx, code int, a *account) {
	exp := time.Now().Add(12 * time.Hour).Unix()
	payload := fmt.Sprintf("%s.%d", base64.RawURLEncoding.EncodeToString([]byte(a.User)), exp)
	c.JSON(code, map[string]any{"token": payload + "." + sign(payload), "user": a.User, "name": a.Name, "expiresAt": exp})
}

type creds struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func main() {
	s := kit.New("auth")
	pg, err := kit.NewPG(os.Getenv("DATABASE_URL"), 4)
	if err != nil {
		log.Fatalf("DATABASE_URL: %v", err)
	}
	db := &store{db: pg, mem: map[string]string{}}

	// A ready-made demo account, so a live session can sign in straight away.
	demoUser, demoPass := "devopsshack", os.Getenv("DEMO_PASSWORD")
	if demoPass == "" {
		demoPass = "devopsshack"
	}
	go func() {
		for i := 0; i < 20; i++ {
			if _, err := db.create(newAccount(demoUser, "DevOps Shack", demoPass)); err == nil {
				return
			}
			time.Sleep(3 * time.Second)
		}
		log.Printf(`{"service":"auth","warn":"could not create the demo account"}`)
	}()

	s.Handle("POST /api/auth/signup", func(c *kit.Ctx) {
		var in creds
		if c.Bind(&in) != nil {
			c.Error(400, "username and password are required")
			return
		}
		user := strings.ToLower(strings.TrimSpace(in.User))
		if !validUser.MatchString(user) {
			c.Error(400, "username: 3–24 characters, letters, numbers, dot, dash or underscore")
			return
		}
		if len(in.Password) < 6 {
			c.Error(400, "password must be at least 6 characters")
			return
		}
		name := strings.TrimSpace(in.Name)
		if name == "" {
			name = user
		}
		a := newAccount(user, name, in.Password)
		ok, err := db.create(a)
		if err != nil {
			c.Error(503, "account store unavailable: "+err.Error())
			return
		}
		if !ok {
			c.Error(409, "that username is already taken")
			return
		}
		issue(c, 201, &a) // signed in straight after sign up
	})

	s.Handle("POST /api/auth/login", func(c *kit.Ctx) {
		var in creds
		if c.Bind(&in) != nil || in.User == "" || in.Password == "" {
			c.Error(400, "username and password are required")
			return
		}
		a, err := db.get(strings.ToLower(strings.TrimSpace(in.User)))
		if err != nil {
			c.Error(503, "account store unavailable: "+err.Error())
			return
		}
		if a == nil || !a.check(in.Password) {
			c.Error(401, "wrong username or password")
			return
		}
		issue(c, 200, a)
	})

	s.Handle("GET /api/auth/verify", func(c *kit.Ctx) {
		tok := c.Query("token")
		if tok == "" {
			tok = c.Token()
		}
		p := strings.Split(tok, ".")
		if len(p) != 3 || !hmac.Equal([]byte(sign(p[0]+"."+p[1])), []byte(p[2])) {
			c.Error(401, "invalid token")
			return
		}
		var exp int64
		fmt.Sscan(p[1], &exp)
		if time.Now().Unix() > exp {
			c.Error(401, "token expired")
			return
		}
		u, _ := base64.RawURLEncoding.DecodeString(p[0])
		c.JSON(200, map[string]any{"valid": true, "user": string(u)})
	})
	s.Run()
}
