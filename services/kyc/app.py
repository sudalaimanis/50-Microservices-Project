# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""kyc — identity verification status (Aadhaar-style, mocked). Booking requires a verified user."""
import hashlib
from railkit import Service

s = Service("kyc")


@s.get("/api/kyc/status/{user}")
def status(c):
    user = c.params["user"]
    blocked = user.startswith("blocked")
    ref = hashlib.sha1(user.encode()).hexdigest()[:8].upper()
    c.json(200, {"user": user, "verified": not blocked, "method": "AADHAAR_OTP",
                 "maskedId": f"XXXX-XXXX-{ref[:4]}", "reference": f"KYC-{ref}"})


s.run()
