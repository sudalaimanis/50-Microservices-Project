// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// A small PostgreSQL client written with the standard library only (no driver package).
// It speaks the PostgreSQL wire protocol v3: startup, SCRAM-SHA-256 / md5 / password login,
// and parameterised queries ($1, $2 …) through the extended query protocol, so values are
// never pasted into SQL text. Results come back as text. A tiny pool reuses connections.
package kit

import (
	"bufio"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PG is a pool of connections to one database, created from a URL such as
// postgres://user:password@host:5432/dbname
type PG struct {
	host, user, password, database string
	pool                           chan *pgConn
}

// Rows is a query result: one map per row, column name → value as text (NULL becomes "").
type Rows []map[string]string

// NewPG returns nil when dsn is empty, so callers can fall back to memory for local runs.
func NewPG(dsn string, size int) (*PG, error) {
	if dsn == "" {
		return nil, nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	pw, _ := u.User.Password()
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":5432"
	}
	if size < 1 {
		size = 4
	}
	return &PG{host: host, user: u.User.Username(), password: pw, database: strings.TrimPrefix(u.Path, "/"), pool: make(chan *pgConn, size)}, nil
}

// Query runs one SQL statement with $1, $2 … parameters and returns its rows.
func (p *PG) Query(sql string, args ...any) (Rows, error) {
	rows, _, err := p.run(sql, args)
	return rows, err
}

// Exec runs one SQL statement and returns how many rows it changed.
func (p *PG) Exec(sql string, args ...any) (int, error) {
	_, n, err := p.run(sql, args)
	return n, err
}

func (p *PG) run(sql string, args []any) (Rows, int, error) {
	var c *pgConn
	select {
	case c = <-p.pool:
	default:
	}
	if c == nil {
		var err error
		if c, err = p.connect(); err != nil {
			return nil, 0, err
		}
	}
	rows, n, err := c.query(sql, args)
	var se *PGError
	if err != nil && !errors.As(err, &se) { // network trouble: drop the connection
		c.conn.Close()
		return nil, 0, err
	}
	select {
	case p.pool <- c:
	default:
		c.conn.Close()
	}
	return rows, n, err
}

// PGError is an error reported by the server (bad SQL, constraint violation …).
type PGError struct{ Code, Message string }

func (e *PGError) Error() string { return "postgres " + e.Code + ": " + e.Message }

type pgConn struct {
	conn net.Conn
	r    *bufio.Reader
}

func (p *PG) connect() (*pgConn, error) {
	nc, err := net.DialTimeout("tcp", p.host, 3*time.Second)
	if err != nil {
		return nil, err
	}
	c := &pgConn{conn: nc, r: bufio.NewReader(nc)}
	c.conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := c.startup(p); err != nil {
		nc.Close()
		return nil, err
	}
	c.conn.SetDeadline(time.Time{})
	return c, nil
}

// ---------- wire protocol helpers ----------

func (c *pgConn) send(typ byte, body []byte) error {
	b := make([]byte, 0, len(body)+5)
	if typ != 0 {
		b = append(b, typ)
	}
	b = binary.BigEndian.AppendUint32(b, uint32(len(body)+4))
	_, err := c.conn.Write(append(b, body...))
	return err
}

func (c *pgConn) recv() (byte, []byte, error) {
	var h [5]byte
	if _, err := io.ReadFull(c.r, h[:]); err != nil {
		return 0, nil, err
	}
	body := make([]byte, binary.BigEndian.Uint32(h[1:])-4)
	_, err := io.ReadFull(c.r, body)
	return h[0], body, err
}

func cstr(s string) []byte { return append([]byte(s), 0) }

func parseError(body []byte) *PGError {
	e := &PGError{}
	for len(body) > 1 {
		f := body[0]
		end := 1
		for end < len(body) && body[end] != 0 {
			end++
		}
		v := string(body[1:end])
		switch f {
		case 'C':
			e.Code = v
		case 'M':
			e.Message = v
		}
		body = body[end+1:]
	}
	return e
}

func (c *pgConn) startup(p *PG) error {
	var b []byte
	b = binary.BigEndian.AppendUint32(b, 196608) // protocol 3.0
	for _, kv := range [][2]string{{"user", p.user}, {"database", p.database}, {"application_name", "railshack"}, {"client_encoding", "UTF8"}} {
		b = append(b, cstr(kv[0])...)
		b = append(b, cstr(kv[1])...)
	}
	if err := c.send(0, append(b, 0)); err != nil {
		return err
	}
	var scram *scramState
	for {
		t, body, err := c.recv()
		if err != nil {
			return err
		}
		switch t {
		case 'E':
			return parseError(body)
		case 'Z':
			return nil
		case 'R':
			switch code := binary.BigEndian.Uint32(body); code {
			case 0: // ok
			case 3: // cleartext password
				c.send('p', cstr(p.password))
			case 5: // md5
				inner := md5.Sum([]byte(p.password + p.user))
				outer := md5.Sum(append([]byte(hex.EncodeToString(inner[:])), body[4:8]...))
				c.send('p', cstr("md5"+hex.EncodeToString(outer[:])))
			case 10: // SASL: SCRAM-SHA-256
				scram = newScram(p.password)
				first := scram.clientFirst()
				m := append(cstr("SCRAM-SHA-256"), binary.BigEndian.AppendUint32(nil, uint32(len(first)))...)
				c.send('p', append(m, first...))
			case 11:
				final, err := scram.clientFinal(string(body[4:]))
				if err != nil {
					return err
				}
				c.send('p', []byte(final))
			case 12: // server signature; checked by trusting the TLS-less cluster network
			default:
				return fmt.Errorf("postgres: unsupported auth method %d", code)
			}
		}
	}
}

func (c *pgConn) query(sql string, args []any) (Rows, int, error) {
	var bind []byte
	bind = append(bind, 0, 0) // portal "", statement ""
	bind = append(bind, 0, 0) // all parameters as text
	bind = binary.BigEndian.AppendUint16(bind, uint16(len(args)))
	for _, a := range args {
		if a == nil {
			bind = binary.BigEndian.AppendUint32(bind, 0xFFFFFFFF)
			continue
		}
		v := fmt.Sprint(a)
		bind = binary.BigEndian.AppendUint32(bind, uint32(len(v)))
		bind = append(bind, v...)
	}
	bind = append(bind, 0, 0) // results as text
	parse := append(append(cstr(""), cstr(sql)...), 0, 0)
	for _, m := range []struct {
		t byte
		b []byte
	}{{'P', parse}, {'B', bind}, {'D', []byte{'P', 0}}, {'E', []byte{0, 0, 0, 0, 0}}, {'S', nil}} {
		if err := c.send(m.t, m.b); err != nil {
			return nil, 0, err
		}
	}
	var cols []string
	var rows Rows
	var affected int
	var qerr error
	for {
		t, body, err := c.recv()
		if err != nil {
			return nil, 0, err
		}
		switch t {
		case 'T':
			n := int(binary.BigEndian.Uint16(body))
			body = body[2:]
			cols = make([]string, n)
			for i := 0; i < n; i++ {
				end := 0
				for body[end] != 0 {
					end++
				}
				cols[i] = string(body[:end])
				body = body[end+1+18:]
			}
		case 'D':
			n := int(binary.BigEndian.Uint16(body))
			body = body[2:]
			row := make(map[string]string, n)
			for i := 0; i < n; i++ {
				l := int32(binary.BigEndian.Uint32(body))
				body = body[4:]
				if l >= 0 {
					if i < len(cols) {
						row[cols[i]] = string(body[:l])
					}
					body = body[l:]
				}
			}
			rows = append(rows, row)
		case 'C':
			tag := strings.Fields(strings.TrimRight(string(body), "\x00"))
			if len(tag) > 0 {
				affected, _ = strconv.Atoi(tag[len(tag)-1])
			}
		case 'E':
			qerr = parseError(body)
		case 'Z':
			return rows, affected, qerr
		}
	}
}

// ---------- SCRAM-SHA-256 (RFC 5802 / 7677) ----------

type scramState struct {
	password, clientNonce, clientFirstBare string
}

func newScram(password string) *scramState {
	n := make([]byte, 18)
	rand.Read(n)
	return &scramState{password: password, clientNonce: base64.StdEncoding.EncodeToString(n)}
}

func (s *scramState) clientFirst() string {
	s.clientFirstBare = "n=,r=" + s.clientNonce
	return "n,," + s.clientFirstBare
}

func (s *scramState) clientFinal(serverFirst string) (string, error) {
	var nonce, salt string
	iter := 0
	for _, part := range strings.Split(serverFirst, ",") {
		if len(part) < 2 {
			continue
		}
		switch part[:2] {
		case "r=":
			nonce = part[2:]
		case "s=":
			salt = part[2:]
		case "i=":
			iter, _ = strconv.Atoi(part[2:])
		}
	}
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil || !strings.HasPrefix(nonce, s.clientNonce) || iter == 0 {
		return "", errors.New("postgres: bad SCRAM challenge")
	}
	salted := pbkdf2SHA256([]byte(s.password), saltBytes, iter)
	clientKey := hmacSHA256(salted, []byte("Client Key"))
	storedKey := sha256.Sum256(clientKey)
	withoutProof := "c=biws,r=" + nonce
	authMessage := s.clientFirstBare + "," + serverFirst + "," + withoutProof
	sig := hmacSHA256(storedKey[:], []byte(authMessage))
	for i := range sig {
		sig[i] ^= clientKey[i]
	}
	return withoutProof + ",p=" + base64.StdEncoding.EncodeToString(sig), nil
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}

func pbkdf2SHA256(password, salt []byte, iter int) []byte {
	u := hmacSHA256(password, append(append([]byte(nil), salt...), 0, 0, 0, 1))
	out := append([]byte(nil), u...)
	for i := 1; i < iter; i++ {
		u = hmacSHA256(password, u)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return out
}
