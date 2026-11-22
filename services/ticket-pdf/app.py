# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""ticket-pdf — renders the e-ticket: a printable layout plus a signed QR payload for TTE checks."""
import hashlib
import json
from railkit import Service, env

s = Service("ticket-pdf")
SECRET = env("TICKET_SECRET", "railshack-ticket")


@s.post("/api/ticket-pdf/render")
def render(c):
    b = c.body or {}
    payload = f"{b.get('pnr')}|{b.get('train')}|{b.get('date')}|{b.get('class')}"
    sig = hashlib.sha256((payload + SECRET).encode()).hexdigest()[:16]
    lines = [
        "RAILSHACK E-TICKET  ·  DevOps Shack",
        f"PNR {b.get('pnr')}   Train {b.get('train')} {b.get('trainName', '')}",
        f"{b.get('from')} → {b.get('to')}   Date {b.get('date')}   Class {b.get('class')}   Quota {b.get('quota')}",
    ] + [f"  {i + 1}. {p.get('name')}  {p.get('label', p.get('status'))}" for i, p in enumerate(b.get("passengers") or [])]
    lines.append(f"Total paid ₹{b.get('total')}")
    c.json(200, {"format": "text/plain", "text": "\n".join(lines), "qr": f"{payload}|{sig}",
                 "downloadName": f"railshack-{b.get('pnr')}.txt", "bytes": len(json.dumps(b))})


s.run()
