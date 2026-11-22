# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""settlements — the finance ledger: every charge and refund, rolled up for reconciliation."""
import threading
import time
from railkit import Service

s = Service("settlements")
lock = threading.Lock()
ledger = []


@s.post("/api/settlements/record")
def record(c):
    e = dict(c.body or {})
    e["at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    with lock:
        ledger.append(e)
        del ledger[:-1000]
    c.json(201, {"recorded": True})


@s.get("/api/settlements/summary")
def summary(c):
    with lock:
        charges = sum(e.get("amount", 0) for e in ledger if e.get("type") == "CHARGE")
        refunds = sum(e.get("amount", 0) for e in ledger if e.get("type") == "REFUND")
        by_method = {}
        for e in ledger:
            by_method[e.get("method")] = by_method.get(e.get("method"), 0) + 1
    c.json(200, {"entries": len(ledger), "charges": round(charges, 2), "refunds": round(refunds, 2),
                 "net": round(charges - refunds, 2), "byMethod": by_method})


s.run()
