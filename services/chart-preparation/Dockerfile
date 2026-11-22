# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
#
# chart-preparation — Python microservice (standard library only). Build from the repo root:
#   docker build -f services/chart-preparation/Dockerfile -t railshack-chart-preparation .
FROM python:3.13-alpine
LABEL org.opencontainers.image.title="railshack-chart-preparation" \
      org.opencontainers.image.description="RailShack chart-preparation microservice · 50 Microservices Project by DevOps Shack" \
      org.opencontainers.image.vendor="DevOps Shack" \
      org.opencontainers.image.authors="DevOps Shack <office@devopsshack.com>" \
      org.opencontainers.image.url="https://devopsshack.com" \
      org.opencontainers.image.source="https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack" \
      org.opencontainers.image.documentation="https://www.youtube.com/@devopsshack" \
      org.opencontainers.image.licenses="Apache-2.0" \
      com.devopsshack.project="RailShack - 50 Microservices Project by DevOps Shack"
WORKDIR /app
ENV PYTHONPATH=/app/lib PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1
COPY NOTICE ./NOTICE
COPY libs/pykit ./lib
COPY services/chart-preparation/app.py ./
USER 1000:1000
EXPOSE 8080
CMD ["python", "app.py"]
