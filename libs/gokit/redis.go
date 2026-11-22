// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

package kit

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// Redis speaks just enough of the RESP protocol for a cache: GET, SET … EX, DEL.
// A nil *Redis (no REDIS_ADDR) is valid and simply never hits, so the cache is optional.
type Redis struct{ Addr string }

func NewRedis(addr string) *Redis {
	if addr == "" {
		return nil
	}
	return &Redis{Addr: addr}
}

func (r *Redis) Do(args ...string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("redis not configured")
	}
	conn, err := net.DialTimeout("tcp", r.Addr, time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Second))
	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(a), a)
	}
	conn.Write([]byte(b.String()))
	rd := bufio.NewReader(conn)
	line, err := rd.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	switch line[0] {
	case '+', ':':
		return line[1:], nil
	case '-':
		return "", fmt.Errorf("redis: %s", line[1:])
	case '$':
		n, _ := strconv.Atoi(line[1:])
		if n < 0 {
			return "", nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(rd, buf); err != nil {
			return "", err
		}
		return string(buf[:n]), nil
	}
	return line, nil
}

// CacheGet returns the cached value and true on a hit. Any Redis problem counts as a miss.
func (r *Redis) CacheGet(key string) (string, bool) {
	v, err := r.Do("GET", key)
	return v, err == nil && v != ""
}

// CacheSet stores a copy for ttl seconds. Failures are ignored: the database still has the data.
func (r *Redis) CacheSet(key, value string, ttl int) {
	r.Do("SET", key, value, "EX", strconv.Itoa(ttl))
}

// CacheDel removes a stale copy after the database changed.
func (r *Redis) CacheDel(key string) { r.Do("DEL", key) }
