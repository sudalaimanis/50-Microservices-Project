# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""railkit — shared runtime for every RailShack Python microservice.

Standard library only. Gives each service health and readiness probes,
Prometheus metrics, JSON logs, request-id propagation, service discovery,
downstream calls with timeouts and the X-Trace header the UI animates.
"""
import hashlib
import json
import os
import random
import re
import signal
import socket
import sys
import threading
import time
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, unquote, urlparse


BRAND = {"project": "RailShack by DevOps Shack", "website": "https://devopsshack.com",
         "youtube": "https://www.youtube.com/@devopsshack",
         "source": "https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack", "license": "Apache-2.0"}
POWERED_BY = "RailShack by DevOps Shack (devopsshack.com)"


def env(k, d=None):
    v = os.environ.get(k)
    return v if v not in (None, "") else d


_PORTS = None
if env("PORTS_FILE"):
    with open(env("PORTS_FILE")) as f:
        _PORTS = {e["name"]: e["port"] for e in json.load(f)}


def pod():
    return env("POD_NAME", socket.gethostname())


def url(svc):
    if _PORTS and svc in _PORTS:
        return f"http://127.0.0.1:{_PORTS[svc]}"
    return env(f"SVC_{svc.upper().replace('-', '_')}_URL", f"http://{svc}:8080")


class CallError(Exception):
    pass


class Ctx:
    def __init__(self, svc, handler, params, query, body):
        self.svc, self.h, self.params, self.query, self.body = svc, handler, params, query, body
        self.trace, self.code = [], 200

    def json(self, code, obj):
        self.code = code
        data = json.dumps(obj).encode()
        trace = ",".join(dict.fromkeys([self.svc.name] + [t for t in self.trace if t]))
        self.h.send_response(code)
        for k, v in (("Content-Type", "application/json"), ("X-Trace", trace), ("X-Pod", pod()), ("X-Powered-By", POWERED_BY),
                     ("Access-Control-Allow-Origin", "*"), ("Access-Control-Expose-Headers", "X-Trace, X-Pod"),
                     ("Content-Length", str(len(data)))):
            self.h.send_header(k, v)
        self.h.end_headers()
        self.h.wfile.write(data)

    def error(self, code, msg):
        self.json(code, {"error": msg, "service": self.svc.name})

    def work(self):
        buf = self.svc.name.encode()
        for _ in range(self.svc.work * 400):
            buf = hashlib.sha256(buf).digest()

    def call(self, svc, method, path, payload=None):
        data = json.dumps(payload).encode() if payload is not None else None
        req = urllib.request.Request(url(svc) + path, data=data, method=method, headers={
            "Content-Type": "application/json", "X-Request-Id": self.h.headers.get("X-Request-Id", "")})
        try:
            with urllib.request.urlopen(req, timeout=self.svc.timeout) as r:
                tr = r.headers.get("X-Trace")
                self.trace += tr.split(",") if tr else [svc]
                text = r.read().decode()
                return json.loads(text) if text else None
        except urllib.error.HTTPError as e:
            self.trace.append(svc)
            raise CallError(f"{svc} returned {e.code}: {e.read().decode()[:200]}")
        except Exception as e:  # noqa: BLE001
            raise CallError(f"{svc} unreachable: {e}")

    def fire(self, svc, method, path, payload=None):
        try:
            self.call(svc, method, path, payload)
        except CallError as e:
            print(json.dumps({"service": self.svc.name, "warn": str(e)}), flush=True)


class Service:
    def __init__(self, name):
        self.name = name
        self.routes = []
        self.started = time.time()
        self.work = int(env("WORK_FACTOR", "0"))
        self.latency = int(env("LATENCY_MS", "0"))
        self.fail_rate = float(env("FAIL_RATE", "0"))
        self.timeout = int(env("CALL_TIMEOUT_MS", "3000")) / 1000
        self.lock = threading.Lock()
        self.reqs, self.dur_sum, self.dur_count = {}, 0.0, 0
        self.get(f"/api/{name}/info")(lambda c: c.json(200, self.info()))

    def info(self):
        return {"service": self.name, "version": env("SERVICE_VERSION", "1.0.0"), "language": "python",
                "pod": pod(), "uptimeSeconds": int(time.time() - self.started), **BRAND}

    def _route(self, method, pattern):
        rx = re.compile("^" + re.sub(r"\{(\w+)\}", r"(?P<\1>[^/]+)", pattern) + "$")

        def deco(fn):
            self.routes.append((method, rx, fn))
            return fn
        return deco

    def get(self, p):
        return self._route("GET", p)

    def post(self, p):
        return self._route("POST", p)

    def metrics_text(self):
        out = ["# TYPE railshack_http_requests_total counter"]
        with self.lock:
            for k, v in sorted(self.reqs.items()):
                m, code = k.split("|")
                out.append(f'railshack_http_requests_total{{service="{self.name}",method="{m}",code="{code}"}} {v}')
            out.append("# TYPE railshack_http_request_duration_seconds summary")
            out.append(f'railshack_http_request_duration_seconds_sum{{service="{self.name}"}} {self.dur_sum:.6f}')
            out.append(f'railshack_http_request_duration_seconds_count{{service="{self.name}"}} {self.dur_count}')
        out.append(f'railshack_up{{service="{self.name}",language="python"}} 1')
        out.append(f'railshack_build_info{{service="{self.name}",version="{env("SERVICE_VERSION", "1.0.0")}",'
                   f'project="{BRAND["project"]}",source="{BRAND["source"]}"}} 1')
        return "\n".join(out) + "\n"

    def run(self):
        svc = self

        class H(BaseHTTPRequestHandler):
            protocol_version = "HTTP/1.1"

            def log_message(self, *a):
                pass

            def _plain(self, text, ctype="text/plain"):
                b = text.encode()
                self.send_response(200)
                self.send_header("Content-Type", ctype)
                self.send_header("Content-Length", str(len(b)))
                self.end_headers()
                self.wfile.write(b)

            def _handle(self, method):
                start = time.time()
                u = urlparse(self.path)
                if u.path == "/healthz":
                    return self._plain("ok")
                if u.path == "/readyz":
                    return self._plain("ready")
                if u.path == "/metrics":
                    return self._plain(svc.metrics_text())
                length = int(self.headers.get("Content-Length") or 0)
                raw = self.rfile.read(length) if length else b""
                try:
                    body = json.loads(raw) if raw else None
                except ValueError:
                    body = None
                query = {k: v[0] for k, v in parse_qs(u.query).items()}
                for m, rx, fn in svc.routes:
                    mt = rx.match(u.path)
                    if m == method and mt:
                        c = Ctx(svc, self, {k: unquote(v) for k, v in mt.groupdict().items()}, query, body)
                        try:
                            if svc.latency:
                                time.sleep(svc.latency / 1000)
                            if svc.fail_rate and random.random() < svc.fail_rate:
                                c.error(500, "injected failure (FAIL_RATE)")
                            else:
                                fn(c)
                        except CallError as e:
                            c.error(502, str(e))
                        except Exception as e:  # noqa: BLE001
                            c.error(500, f"{type(e).__name__}: {e}")
                        break
                else:
                    c = Ctx(svc, self, {}, query, body)
                    c.error(404, "not found")
                d = time.time() - start
                with svc.lock:
                    k = f"{method}|{c.code}"
                    svc.reqs[k] = svc.reqs.get(k, 0) + 1
                    svc.dur_sum += d
                    svc.dur_count += 1
                print(json.dumps({"ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "service": svc.name,
                                  "method": method, "path": u.path, "status": c.code, "ms": int(d * 1000),
                                  "requestId": self.headers.get("X-Request-Id", "")}), flush=True)

            def do_GET(self):
                self._handle("GET")

            def do_POST(self):
                self._handle("POST")

        port = int(env("PORT", "8080"))
        srv = ThreadingHTTPServer(("0.0.0.0", port), H)
        srv.daemon_threads = True
        signal.signal(signal.SIGTERM, lambda *a: (threading.Thread(target=srv.shutdown).start()))
        print(json.dumps({"service": self.name, "msg": "listening", "port": port, "language": "python",
                          "project": BRAND["project"], "youtube": BRAND["youtube"]}), flush=True)
        try:
            srv.serve_forever()
        except KeyboardInterrupt:
            pass
        sys.exit(0)
