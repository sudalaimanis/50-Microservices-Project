# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
#
# fares — Java 21 microservice (JDK only, no framework). Build from the repo root:
#   docker build -f services/fares/Dockerfile -t railshack-fares .
# Bytecode is platform-independent, so it compiles once on the build machine.
FROM --platform=$BUILDPLATFORM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /src
COPY libs/javakit/Kit.java services/fares/Main.java ./
RUN javac -d /out Kit.java Main.java

FROM eclipse-temurin:21-jre
LABEL org.opencontainers.image.title="railshack-fares" \
      org.opencontainers.image.description="RailShack fares microservice · 50 Microservices Project by DevOps Shack" \
      org.opencontainers.image.vendor="DevOps Shack" \
      org.opencontainers.image.authors="DevOps Shack <office@devopsshack.com>" \
      org.opencontainers.image.url="https://devopsshack.com" \
      org.opencontainers.image.source="https://github.com/jaiswaladi246/50-MicroServices-Project-By-DevOpsShack" \
      org.opencontainers.image.documentation="https://www.youtube.com/@devopsshack" \
      org.opencontainers.image.licenses="Apache-2.0" \
      com.devopsshack.project="RailShack - 50 Microservices Project by DevOps Shack"
WORKDIR /app
COPY NOTICE ./NOTICE
COPY --from=build /out ./
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "-XX:+UseSerialGC", "-cp", "/app", "Main"]
