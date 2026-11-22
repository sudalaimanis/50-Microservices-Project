# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
#
# search — Go microservice. Build from the repo root:
#   docker build -f services/search/Dockerfile -t railshack-search .
# Cross-compiles on the build machine, so amd64 and arm64 images build fast without emulation.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY libs/gokit ./libs/gokit
COPY services/search ./services/search
WORKDIR /src/services/search
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opencontainers.image.title="railshack-search" \
      org.opencontainers.image.description="RailShack search microservice · 50 Microservices Project by DevOps Shack" \
      org.opencontainers.image.vendor="DevOps Shack" \
      org.opencontainers.image.authors="DevOps Shack <office@devopsshack.com>" \
      org.opencontainers.image.url="https://devopsshack.com" \
      org.opencontainers.image.source="https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack" \
      org.opencontainers.image.documentation="https://www.youtube.com/@devopsshack" \
      org.opencontainers.image.licenses="Apache-2.0" \
      com.devopsshack.project="RailShack - 50 Microservices Project by DevOps Shack"
COPY NOTICE /NOTICE
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
