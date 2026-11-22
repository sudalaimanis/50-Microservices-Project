# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""chart-preparation — the final reservation chart is prepared four hours before departure."""
import datetime as dt
from railkit import Service

s = Service("chart-preparation")
IST = dt.timezone(dt.timedelta(hours=5, minutes=30))


@s.get("/api/chart-preparation/status")
def status(c):
    train, date = c.query.get("train", ""), c.query.get("date", "")
    sched = c.call("schedules", "GET", f"/api/schedules/{train}")
    try:
        h, m = map(int, sched.get("departs", "00:00").split(":"))
        dep = dt.datetime.combine(dt.date.fromisoformat(date), dt.time(h, m), IST)
    except ValueError:
        return c.error(400, "date must be YYYY-MM-DD")
    chart_at = dep - dt.timedelta(hours=4)
    prepared = dt.datetime.now(IST) >= chart_at
    c.json(200, {"train": train, "date": date, "prepared": prepared,
                 "chartTime": chart_at.strftime("%d %b %H:%M IST"),
                 "message": "Chart prepared" if prepared else f"Chart will be prepared at {chart_at.strftime('%d %b %H:%M')}"})


s.run()
