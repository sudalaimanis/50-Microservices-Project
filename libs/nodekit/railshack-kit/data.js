// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// Data clients written with Node's built-in modules only (no npm packages):
//   pg(url)     a small PostgreSQL client: SCRAM-SHA-256 / md5 / password login and
//               parameterised queries ($1, $2 …) through the extended query protocol.
//   redis(addr) just enough RESP for a cache: GET, SET … EX, DEL. Any Redis problem is a cache miss.
'use strict';
const net = require('net');
const crypto = require('crypto');

const cstr = (s) => Buffer.from(s + '\0');
function pgError(b) {
  const f = {};
  let i = 0;
  while (i < b.length && b[i] !== 0) { const end = b.indexOf(0, i + 1); f[String.fromCharCode(b[i])] = b.toString('utf8', i + 1, end); i = end + 1; }
  const e = new Error(`postgres ${f.C}: ${f.M}`); e.code = f.C; e.pg = true; return e;
}

class Conn {
  constructor(sock) {
    this.sock = sock; this.buf = Buffer.alloc(0); this.msgs = []; this.waiters = []; this.dead = null;
    sock.on('data', (d) => {
      this.buf = Buffer.concat([this.buf, d]);
      while (this.buf.length >= 5) {
        const len = this.buf.readUInt32BE(1);
        if (this.buf.length < len + 1) break;
        const m = { t: String.fromCharCode(this.buf[0]), b: this.buf.subarray(5, len + 1) };
        this.buf = this.buf.subarray(len + 1);
        const w = this.waiters.shift(); if (w) w.resolve(m); else this.msgs.push(m);
      }
    });
    const fail = (e) => { this.dead = e || new Error('postgres connection closed'); this.waiters.splice(0).forEach((w) => w.reject(this.dead)); };
    sock.on('error', fail); sock.on('close', () => fail());
  }
  next() {
    if (this.msgs.length) return Promise.resolve(this.msgs.shift());
    if (this.dead) return Promise.reject(this.dead);
    return new Promise((resolve, reject) => this.waiters.push({ resolve, reject }));
  }
  send(t, body) {
    const h = Buffer.alloc(t ? 5 : 4);
    if (t) h[0] = t.charCodeAt(0);
    h.writeUInt32BE(body.length + 4, t ? 1 : 0);
    this.sock.write(Buffer.concat([h, body]));
  }
}

function pg(dsn) {
  if (!dsn) return null;
  const u = new URL(dsn);
  const cfg = { host: u.hostname, port: Number(u.port || 5432), user: decodeURIComponent(u.username), password: decodeURIComponent(u.password), database: u.pathname.slice(1) };
  let conn = null, chain = Promise.resolve();

  async function connect() {
    const sock = net.createConnection({ host: cfg.host, port: cfg.port });
    await new Promise((res, rej) => { sock.once('connect', res); sock.once('error', rej); sock.setTimeout(5000, () => rej(new Error('postgres connect timeout'))); });
    sock.setTimeout(0);
    const c = new Conn(sock);
    const ver = Buffer.alloc(4); ver.writeUInt32BE(196608);
    c.send('', Buffer.concat([ver, cstr('user'), cstr(cfg.user), cstr('database'), cstr(cfg.database), cstr('client_encoding'), cstr('UTF8'), Buffer.from([0])]));
    let nonce, firstBare;
    for (;;) {
      const m = await c.next();
      if (m.t === 'E') throw pgError(m.b);
      if (m.t === 'Z') return c;
      if (m.t !== 'R') continue;
      const code = m.b.readUInt32BE(0);
      if (code === 3) c.send('p', cstr(cfg.password));
      else if (code === 5) {
        const inner = crypto.createHash('md5').update(cfg.password + cfg.user).digest('hex');
        c.send('p', cstr('md5' + crypto.createHash('md5').update(Buffer.concat([Buffer.from(inner), m.b.subarray(4, 8)])).digest('hex')));
      } else if (code === 10) {
        nonce = crypto.randomBytes(18).toString('base64'); firstBare = 'n=,r=' + nonce;
        const first = Buffer.from('n,,' + firstBare), len = Buffer.alloc(4); len.writeUInt32BE(first.length);
        c.send('p', Buffer.concat([cstr('SCRAM-SHA-256'), len, first]));
      } else if (code === 11) {
        const serverFirst = m.b.subarray(4).toString();
        const p = Object.fromEntries(serverFirst.split(',').map((kv) => [kv[0], kv.slice(2)]));
        if (!p.r.startsWith(nonce)) throw new Error('postgres: bad SCRAM nonce');
        const salted = crypto.pbkdf2Sync(cfg.password, Buffer.from(p.s, 'base64'), Number(p.i), 32, 'sha256');
        const clientKey = crypto.createHmac('sha256', salted).update('Client Key').digest();
        const storedKey = crypto.createHash('sha256').update(clientKey).digest();
        const without = 'c=biws,r=' + p.r;
        const sig = crypto.createHmac('sha256', storedKey).update(`${firstBare},${serverFirst},${without}`).digest();
        for (let i = 0; i < sig.length; i++) sig[i] ^= clientKey[i];
        c.send('p', Buffer.from(`${without},p=${sig.toString('base64')}`));
      } else if (code !== 0 && code !== 12) throw new Error('postgres: unsupported auth method ' + code);
    }
  }

  async function run(sql, params) {
    if (!conn || conn.dead) conn = await connect();
    const c = conn;
    const parts = [Buffer.from([0, 0, 0, 0]), Buffer.alloc(2)];
    parts[1].writeUInt16BE(params.length);
    for (const v of params) {
      if (v === null || v === undefined) { const n = Buffer.alloc(4); n.writeInt32BE(-1); parts.push(n); continue; }
      const b = Buffer.from(typeof v === 'string' ? v : String(v)), l = Buffer.alloc(4); l.writeUInt32BE(b.length); parts.push(l, b);
    }
    parts.push(Buffer.from([0, 0]));
    c.send('P', Buffer.concat([cstr(''), cstr(sql), Buffer.from([0, 0])]));
    c.send('B', Buffer.concat(parts));
    c.send('D', Buffer.from([0x50, 0]));
    c.send('E', Buffer.from([0, 0, 0, 0, 0]));
    c.send('S', Buffer.alloc(0));
    let cols = [], rows = [], affected = 0, err = null;
    for (;;) {
      const m = await c.next();
      if (m.t === 'T') {
        const n = m.b.readUInt16BE(0); let i = 2; cols = [];
        for (let k = 0; k < n; k++) { const end = m.b.indexOf(0, i); cols.push(m.b.toString('utf8', i, end)); i = end + 1 + 18; }
      } else if (m.t === 'D') {
        const n = m.b.readUInt16BE(0); let i = 2; const row = {};
        for (let k = 0; k < n; k++) { const l = m.b.readInt32BE(i); i += 4; row[cols[k]] = l < 0 ? null : m.b.toString('utf8', i, i + l); if (l > 0) i += l; }
        rows.push(row);
      } else if (m.t === 'C') { const tag = m.b.toString('utf8', 0, m.b.length - 1).split(' '); affected = Number(tag[tag.length - 1]) || 0; }
      else if (m.t === 'E') err = pgError(m.b);
      else if (m.t === 'Z') { if (err) throw err; return { rows, affected }; }
    }
  }

  // One query at a time on one connection; a broken connection is replaced on the next query.
  function query(sql, params = []) {
    const p = chain.then(() => run(sql, params)).catch((e) => { if (!e.pg && conn) { conn.sock.destroy(); conn = null; } throw e; });
    chain = p.catch(() => {});
    return p;
  }
  return { query };
}

function redis(addr) {
  if (!addr) return null;
  const [host, port] = addr.split(':');
  function cmd(...args) {
    return new Promise((resolve, reject) => {
      const sock = net.createConnection({ host, port: Number(port || 6379) });
      let buf = Buffer.alloc(0);
      sock.setTimeout(1000, () => { sock.destroy(); reject(new Error('redis timeout')); });
      sock.on('error', reject);
      sock.on('data', (d) => {
        buf = Buffer.concat([buf, d]);
        const end = buf.indexOf('\r\n');
        if (end < 0) return;
        const t = String.fromCharCode(buf[0]), line = buf.toString('utf8', 1, end);
        if (t === '-') { sock.end(); return reject(new Error(line)); }
        if (t !== '$') { sock.end(); return resolve(line); }
        const n = Number(line);
        if (n < 0) { sock.end(); return resolve(null); }
        if (buf.length >= end + 2 + n + 2) { sock.end(); resolve(buf.toString('utf8', end + 2, end + 2 + n)); }
      });
      sock.write(`*${args.length}\r\n` + args.map((a) => `$${Buffer.byteLength(String(a))}\r\n${a}\r\n`).join(''));
    });
  }
  return {
    async get(key) { try { return await cmd('GET', key); } catch { return null; } },
    async set(key, value, ttl) { try { await cmd('SET', key, value, 'EX', String(ttl)); } catch { /* the database still has it */ } },
    async del(key) { try { await cmd('DEL', key); } catch { /* expires on its own */ } },
  };
}

module.exports = { pg, redis };
