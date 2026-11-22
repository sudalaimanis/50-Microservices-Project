#!/usr/bin/env python3
"""Generates the per-service Dockerfiles from services.json.

services.json is the single source of truth for the 50 microservices. When we add
a service, we add one line there, write its code, and re-run:  python3 scripts/generate.py
"""
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent.parent
SERVICES = json.loads((ROOT / "services.json").read_text())

HEADER = """# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
#
"""

LABELS = """LABEL org.opencontainers.image.title="railshack-{name}" \\
      org.opencontainers.image.description="RailShack {name} microservice · 50 Microservices Project by DevOps Shack" \\
      org.opencontainers.image.vendor="DevOps Shack" \\
      org.opencontainers.image.authors="DevOps Shack <office@devopsshack.com>" \\
      org.opencontainers.image.url="https://devopsshack.com" \\
      org.opencontainers.image.source="https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack" \\
      org.opencontainers.image.documentation="https://www.youtube.com/@devopsshack" \\
      org.opencontainers.image.licenses="Apache-2.0" \\
      com.devopsshack.project="RailShack - 50 Microservices Project by DevOps Shack"
"""

DOCKERFILES = {
    "go": HEADER + """# {name} — Go microservice. Build from the repo root:
#   docker build -f services/{name}/Dockerfile -t railshack-{name} .
# Cross-compiles on the build machine, so amd64 and arm64 images build fast without emulation.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY libs/gokit ./libs/gokit
COPY services/{name} ./services/{name}
WORKDIR /src/services/{name}
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot
""" + LABELS + """COPY NOTICE /NOTICE
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
""",
    "node": HEADER + """# {name} — Node.js microservice (zero npm dependencies). Build from the repo root:
#   docker build -f services/{name}/Dockerfile -t railshack-{name} .
FROM node:22-alpine
""" + LABELS + """WORKDIR /app
ENV NODE_ENV=production NODE_PATH=/app/lib
COPY NOTICE ./NOTICE
COPY libs/nodekit ./lib
{extra}COPY services/{name}/ ./
USER node
EXPOSE 8080
CMD ["node", "index.js"]
""",
    "python": HEADER + """# {name} — Python microservice (standard library only). Build from the repo root:
#   docker build -f services/{name}/Dockerfile -t railshack-{name} .
FROM python:3.13-alpine
""" + LABELS + """WORKDIR /app
ENV PYTHONPATH=/app/lib PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1
COPY NOTICE ./NOTICE
COPY libs/pykit ./lib
COPY services/{name}/app.py ./
USER 1000:1000
EXPOSE 8080
CMD ["python", "app.py"]
""",
    "java": HEADER + """# {name} — Java 21 microservice (JDK only, no framework). Build from the repo root:
#   docker build -f services/{name}/Dockerfile -t railshack-{name} .
# Bytecode is platform-independent, so it compiles once on the build machine.
FROM --platform=$BUILDPLATFORM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /src
COPY libs/javakit/Kit.java services/{name}/Main.java ./
RUN javac -d /out Kit.java Main.java

FROM eclipse-temurin:21-jre
""" + LABELS + """WORKDIR /app
COPY NOTICE ./NOTICE
COPY --from=build /out ./
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "-XX:+UseSerialGC", "-cp", "/app", "Main"]
""",
}



def main():
    for s in SERVICES:
        extra = "COPY services.json ./services.json\n" if s["name"] == "frontend" else ""
        df = DOCKERFILES[s["lang"]].format(name=s["name"], extra=extra)
        (ROOT / "services" / s["name"] / "Dockerfile").write_text(df)

    print(f"generated {len(SERVICES)} Dockerfiles")


if __name__ == "__main__":
    main()
