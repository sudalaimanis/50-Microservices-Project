# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""tatkal — Tatkal booking window rules: opens the day before travel, 10:00 for AC and 11:00 for non-AC.

Set TATKAL_ALWAYS_OPEN=true (the default) so live demos never hit a closed window.
"""
import datetime as dt
from railkit import Service, env

s = Service("tatkal")
IST = dt.timezone(dt.timedelta(hours=5, minutes=30))
AC = {"1A", "2A", "3A", "CC", "EC"}


@s.get("/api/tatkal/check")
def check(c):
    cls, date = c.query.get("class", "SL"), c.query.get("date", "")
    opens = "10:00" if cls in AC else "11:00"
    surcharge = {"1A": 0, "2A": 500, "3A": 400, "CC": 225, "EC": 400, "SL": 200, "2S": 15}.get(cls, 200)
    rule = f"Opens one day before travel at {opens} IST"
    if env("TATKAL_ALWAYS_OPEN", "true") == "true":
        return c.json(200, {"open": True, "surcharge": surcharge, "rule": rule, "message": "Tatkal open (demo mode)"})
    try:
        travel = dt.date.fromisoformat(date)
    except ValueError:
        return c.error(400, "date must be YYYY-MM-DD")
    now = dt.datetime.now(IST)
    h, m = map(int, opens.split(":"))
    open_at = dt.datetime.combine(travel - dt.timedelta(days=1), dt.time(h, m), IST)
    is_open = open_at <= now < dt.datetime.combine(travel, dt.time(0, 0), IST) + dt.timedelta(days=1)
    c.json(200, {"open": is_open, "surcharge": surcharge, "rule": rule,
                 "message": "Tatkal open" if is_open else f"Tatkal for {date} {rule.lower()}"})


s.run()
