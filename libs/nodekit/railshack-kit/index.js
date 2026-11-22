// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// railshack-kit — the shared runtime for every RailShack Node.js microservice.
// Zero dependencies: only Node's built-in modules. Gives each service health
// and readiness probes, Prometheus metrics, JSON logs, request-id propagation,
// service discovery, downstream calls and the X-Trace header the UI animates.
'use strict';
const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');

const BRAND = { project: 'RailShack by DevOps Shack', website: 'https://devopsshack.com',
  youtube: 'https://www.youtube.com/@devopsshack', source: 'https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack', license: 'Apache-2.0' };
const POWERED_BY = 'RailShack by DevOps Shack (devopsshack.com)';
const env = (k, d) => (process.env[k] !== undefined && process.env[k] !== '' ? process.env[k] : d);
let ports = null;
if (process.env.PORTS_FILE) {
  ports = {};
  for (const e of JSON.parse(fs.readFileSync(process.env.PORTS_FILE, 'utf8'))) ports[e.name] = e.port;
}

const pod = () => env('POD_NAME', os.hostname());

function url(svc) {
  if (ports && ports[svc]) return `http://127.0.0.1:${ports[svc]}`;
  const o = process.env[`SVC_${svc.toUpperCase().replace(/-/g, '_')}_URL`];
  return o || `http://${svc}:8080`;
}

function dedupe(list) { return [...new Set(list.filter(Boolean))]; }

const MIME = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript', '.css': 'text/css',
  '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon', '.json': 'application/json' };

function service(name) {
  const routes = [];
  const started = Date.now();
  const metrics = { reqs: {}, durSum: 0, durCount: 0 };
  const work = Number(env('WORK_FACTOR', 0));
  const latency = Number(env('LATENCY_MS', 0));
  const failRate = Number(env('FAIL_RATE', 0));
  const timeout = Number(env('CALL_TIMEOUT_MS', 3000));
  let staticDir = null;
  const proxies = [];

  const info = () => ({ service: name, version: env('SERVICE_VERSION', '1.0.0'), language: 'node', pod: pod(),
    uptimeSeconds: Math.round((Date.now() - started) / 1000), ...BRAND });

  function add(method, pattern, fn) {
    const parts = pattern.split('/').filter(Boolean);
    routes.push({ method, parts, fn });
  }

  function match(method, pathname) {
    const segs = pathname.split('/').filter(Boolean);
    for (const r of routes) {
      if (r.method !== method || r.parts.length !== segs.length) continue;
      const params = {};
      let ok = true;
      for (let i = 0; i < segs.length; i++) {
        if (r.parts[i].startsWith(':')) params[r.parts[i].slice(1)] = decodeURIComponent(segs[i]);
        else if (r.parts[i] !== segs[i]) { ok = false; break; }
      }
      if (ok) return { r, params };
    }
    return null;
  }

  function makeCtx(req, res, u, params, body) {
    const c = {
      req, res, params, body, query: Object.fromEntries(u.searchParams), trace: [], code: 200, service: name,
      json(code, obj) {
        c.code = code;
        res.writeHead(code, { 'Content-Type': 'application/json', 'X-Trace': dedupe([name, ...c.trace]).join(','),
          'X-Pod': pod(), 'X-Powered-By': POWERED_BY, 'Access-Control-Allow-Origin': '*', 'Access-Control-Expose-Headers': 'X-Trace, X-Pod' });
        res.end(JSON.stringify(obj));
      },
      error(code, msg) { c.json(code, { error: msg, service: name }); },
      work() {
        if (work <= 0) return;
        let buf = Buffer.from(name);
        for (let i = 0; i < work * 1500; i++) buf = crypto.createHash('sha256').update(buf).digest();
      },
      async call(svc, method, p, payload) {
        const ctl = new AbortController();
        const t = setTimeout(() => ctl.abort(), timeout);
        try {
          const r = await fetch(url(svc) + p, { method, signal: ctl.signal,
            headers: { 'Content-Type': 'application/json', 'X-Request-Id': req.headers['x-request-id'] || '' },
            body: payload !== undefined ? JSON.stringify(payload) : undefined });
          const tr = r.headers.get('x-trace');
          c.trace.push(...(tr ? tr.split(',') : [svc]));
          const text = await r.text();
          if (r.status >= 400) throw new Error(`${svc} returned ${r.status}: ${text}`);
          return text ? JSON.parse(text) : null;
        } catch (e) {
          if (e.name === 'AbortError') throw new Error(`${svc} timed out`);
          if (e.cause) throw new Error(`${svc} unreachable: ${e.cause.code || e.message}`);
          throw e;
        } finally { clearTimeout(t); }
      },
      // token() reads "Authorization: Bearer <token>"; whoami() asks auth who it belongs to (401 if nobody).
      token() {
        const h = req.headers.authorization || '';
        return /^bearer /i.test(h) ? h.slice(7).trim() : '';
      },
      async whoami() {
        const t = c.token();
        if (!t) { const e = new Error('please sign in'); e.status = 401; throw e; }
        try { return (await c.call('auth', 'GET', '/api/auth/verify?token=' + encodeURIComponent(t))).user; }
        catch (err) { const e = new Error(/401/.test(err.message) ? 'your session has expired, please sign in again' : err.message); e.status = /401/.test(err.message) ? 401 : 503; throw e; }
      },
      async fire(svc, method, p, payload) {
        try { await c.call(svc, method, p, payload); } catch (e) { console.log(JSON.stringify({ service: name, warn: e.message })); }
      },
    };
    return c;
  }

  function promText() {
    let out = '# TYPE railshack_http_requests_total counter\n';
    for (const [k, v] of Object.entries(metrics.reqs).sort()) {
      const [m, code] = k.split('|');
      out += `railshack_http_requests_total{service="${name}",method="${m}",code="${code}"} ${v}\n`;
    }
    out += '# TYPE railshack_http_request_duration_seconds summary\n';
    out += `railshack_http_request_duration_seconds_sum{service="${name}"} ${metrics.durSum.toFixed(6)}\n`;
    out += `railshack_http_request_duration_seconds_count{service="${name}"} ${metrics.durCount}\n`;
    out += `railshack_up{service="${name}",language="node"} 1\n`;
    out += `railshack_build_info{service="${name}",version="${env('SERVICE_VERSION', '1.0.0')}",project="${BRAND.project}",source="${BRAND.source}"} 1\n`;
    return out;
  }

  async function proxy(req, res, u, target) {
    const chunks = [];
    for await (const ch of req) chunks.push(ch);
    try {
      const r = await fetch(url(target) + u.pathname + u.search, { method: req.method,
        headers: { 'Content-Type': 'application/json', 'X-Request-Id': req.headers['x-request-id'] || crypto.randomUUID(),
          ...(req.headers.authorization ? { Authorization: req.headers.authorization } : {}) },
        body: ['GET', 'HEAD'].includes(req.method) ? undefined : Buffer.concat(chunks) });
      const body = Buffer.from(await r.arrayBuffer());
      const headers = { 'Content-Type': r.headers.get('content-type') || 'application/json',
        'Access-Control-Expose-Headers': 'X-Trace, X-Pod' };
      if (r.headers.get('x-trace')) headers['X-Trace'] = `${name},${r.headers.get('x-trace')}`;
      if (r.headers.get('x-pod')) headers['X-Pod'] = r.headers.get('x-pod');
      res.writeHead(r.status, headers);
      res.end(body);
    } catch (e) {
      res.writeHead(502, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: `${target} unreachable`, service: name }));
    }
  }

  function serveStatic(res, pathname) {
    let file = path.join(staticDir, path.normalize(pathname).replace(/^(\.\.[/\\])+/, ''));
    if (!file.startsWith(staticDir) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) file = path.join(staticDir, 'index.html');
    res.writeHead(200, { 'Content-Type': MIME[path.extname(file)] || 'application/octet-stream', 'Cache-Control': 'no-cache', 'X-Powered-By': POWERED_BY });
    fs.createReadStream(file).pipe(res);
  }

  async function handler(req, res) {
    const start = process.hrtime.bigint();
    const u = new URL(req.url, 'http://x');
    if (req.method === 'OPTIONS') {
      res.writeHead(204, { 'Access-Control-Allow-Origin': '*', 'Access-Control-Allow-Headers': '*', 'Access-Control-Allow-Methods': '*' });
      return res.end();
    }
    if (u.pathname === '/healthz') return res.end('ok');
    if (u.pathname === '/readyz') return res.end('ready');
    if (u.pathname === '/metrics') { res.writeHead(200, { 'Content-Type': 'text/plain' }); return res.end(promText()); }
    const m = match(req.method, u.pathname);
    if (!m) {
      for (const p of proxies) if (u.pathname.startsWith(p.prefix)) return proxy(req, res, u, p.target);
      if (staticDir && req.method === 'GET') return serveStatic(res, u.pathname);
      res.writeHead(404, { 'Content-Type': 'application/json' });
      return res.end(JSON.stringify({ error: 'not found', service: name }));
    }
    let body = null;
    const chunks = [];
    for await (const ch of req) chunks.push(ch);
    if (chunks.length) { try { body = JSON.parse(Buffer.concat(chunks).toString()); } catch { body = null; } }
    const c = makeCtx(req, res, u, m.params, body);
    try {
      if (latency > 0) await new Promise((r) => setTimeout(r, latency));
      if (failRate > 0 && Math.random() < failRate) c.error(500, 'injected failure (FAIL_RATE)');
      else await m.r.fn(c);
    } catch (e) {
      if (!res.headersSent) c.error(e.status || 502, e.message);
    }
    const ms = Number(process.hrtime.bigint() - start) / 1e6;
    const key = `${req.method}|${c.code}`;
    metrics.reqs[key] = (metrics.reqs[key] || 0) + 1;
    metrics.durSum += ms / 1000;
    metrics.durCount++;
    console.log(JSON.stringify({ ts: new Date().toISOString(), service: name, method: req.method, path: u.pathname,
      status: c.code, ms: Math.round(ms), requestId: req.headers['x-request-id'] || '' }));
  }

  const api = {
    name, url, info,
    get: (p, fn) => add('GET', p, fn),
    post: (p, fn) => add('POST', p, fn),
    put: (p, fn) => add('PUT', p, fn),
    delete: (p, fn) => add('DELETE', p, fn),
    static: (dir) => { staticDir = path.resolve(dir); },
    proxy: (prefix, target) => proxies.push({ prefix, target }),
    run() {
      const port = Number(env('PORT', 8080));
      const srv = http.createServer(handler);
      srv.listen(port, () => console.log(JSON.stringify({ service: name, msg: 'listening', port, language: 'node', project: BRAND.project, youtube: BRAND.youtube })));
      const stop = () => srv.close(() => process.exit(0));
      process.on('SIGTERM', stop);
      process.on('SIGINT', stop);
    },
  };
  api.get(`/api/${name}/info`, (c) => c.json(200, info()));
  return api;
}

const { pg, redis } = require('./data');
module.exports = { service, url, pod, BRAND, pg, redis };
