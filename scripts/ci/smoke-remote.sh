#!/usr/bin/env bash
# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
# Finds the public address of the release (the ALB Ingress on EKS, or a LoadBalancer Service),
# waits until it answers, then runs the end-to-end smoke test through it.
#
# With APP_DOMAIN set, the test talks to our domain (https when ACM_CERT_ARN is set) but connects
# straight to the new ALB (curl --connect-to). So it tests the real certificate and host rules even
# before the DNS record points at the load balancer.
set -euo pipefail
NS="${1:-railshack}"
DOMAIN="${APP_DOMAIN:-}"
SCHEME=http; PORT=80
[ -n "$DOMAIN" ] && [ -n "${ACM_CERT_ARN:-}" ] && { SCHEME=https; PORT=443; }

address() {
  if kubectl get ingress railshack -n "$NS" >/dev/null 2>&1; then
    kubectl get ingress railshack -n "$NS" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
  else
    kubectl get svc frontend -n "$NS" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}{.status.loadBalancer.ingress[0].ip}'
  fi
}
probe() {   # probe <host> <path>: reach <host> through the ALB
  if [ -n "$DOMAIN" ]; then curl -sf --max-time 5 --connect-to "$1:$PORT:$ADDR:$PORT" "$SCHEME://$1$2" >/dev/null
  else curl -sf --max-time 5 "http://$ADDR$2" >/dev/null; fi
}

ADDR=""
for i in $(seq 1 60); do      # up to 10 minutes: a new ALB plus its DNS can take a few minutes
  ADDR="$(address || true)"
  [ -n "$ADDR" ] && probe "$DOMAIN" /healthz && break
  echo "waiting for the public address ($i/60)... ${ADDR:-not assigned yet}"; sleep 10
done
probe "$DOMAIN" /healthz || { echo "public address never became reachable"; exit 1; }

if [ -n "$DOMAIN" ]; then
  URL="$SCHEME://$DOMAIN"
  export CURL_HOME; CURL_HOME="$(mktemp -d)"      # every curl in the smoke test connects to the ALB
  printf 'connect-to = "%s:%s:%s:%s"\nconnect-to = "grafana.%s:%s:%s:%s"\n' \
    "$DOMAIN" "$PORT" "$ADDR" "$PORT" "$DOMAIN" "$PORT" "$ADDR" "$PORT" > "$CURL_HOME/.curlrc"
  if getent hosts "$DOMAIN" >/dev/null; then
    echo "DNS: $DOMAIN resolves."
  else
    echo "::warning::DNS: $DOMAIN does not resolve yet. Create a record for $DOMAIN and grafana.$DOMAIN pointing to $ADDR"
  fi
  if kubectl get ingress railshack-grafana -n monitoring >/dev/null 2>&1; then
    for i in $(seq 1 30); do probe "grafana.$DOMAIN" /api/health && break; echo "waiting for Grafana ($i/30)..."; sleep 10; done
    probe "grafana.$DOMAIN" /api/health && echo "📊 Grafana is live at $SCHEME://grafana.$DOMAIN" || { echo "Grafana is not reachable through the ALB"; exit 1; }
    echo "### 📊 Grafana → $SCHEME://grafana.$DOMAIN" >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
  fi
else
  URL="http://$ADDR"
fi
echo "🌐 RailShack is live at $URL   (load balancer: $ADDR)"
echo "### 🚆 RailShack deployed → $URL" >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
"$(dirname "$0")/../smoke-test.sh" "$URL"
