# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""retiring-rooms — book a room at the station for a layover."""
from railkit import Service

s = Service("retiring-rooms")
ROOMS = [("Dormitory bed", 250, 1), ("Non-AC double", 700, 2), ("AC double", 1200, 2), ("AC deluxe suite", 2200, 3)]


@s.get("/api/retiring-rooms/availability")
def availability(c):
    st = c.call("stations", "GET", f"/api/stations/{c.query.get('station', 'NDLS')}")
    seed = sum(map(ord, st["code"]))
    rooms = [{"type": t, "pricePerNight": p, "guests": g, "available": (seed * (i + 3)) % 9}
             for i, (t, p, g) in enumerate(ROOMS)]
    c.json(200, {"station": st["code"], "stationName": st["name"], "rooms": rooms})


s.run()
