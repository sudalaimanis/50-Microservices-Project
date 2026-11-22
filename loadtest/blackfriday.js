// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// Black Friday load test (k6). Simulates a sale: thousands of people searching trains,
// some of them booking. Every search fans out to ~10 microservices, every booking to ~30.
//   Local:      k6 run -e BASE_URL=http://localhost:9050 -e PEAK_VUS=20 -e HOLD=30s loadtest/blackfriday.js
//   In-cluster: see loadtest/k6-job.yaml
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE = __ENV.BASE_URL || 'http://frontend:8080';
const PEAK = Number(__ENV.PEAK_VUS || 300);
const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };

export const options = {
  scenarios: {
    sale: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: __ENV.WARMUP || '1m', target: Math.ceil(PEAK * 0.2) }, // doors open
        { duration: __ENV.RAMP || '2m', target: PEAK },                     // the rush
        { duration: __ENV.HOLD || '6m', target: PEAK },                     // peak sale
        { duration: __ENV.COOLDOWN || '2m', target: 0 },                    // sale ends
      ],
      gracefulRampDown: '20s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    'http_req_duration{name:search}': ['p(95)<2500'],
  },
};

const ROUTES = [['NDLS', 'BSB'], ['BCT', 'NDLS'], ['HWH', 'NDLS'], ['NDLS', 'JP'], ['ADI', 'NDLS'],
  ['SBC', 'NDLS'], ['NDLS', 'ASR'], ['PNBE', 'NDLS'], ['NDLS', 'LKO'], ['NDLS', 'HWH'], ['BSB', 'NDLS']];

export function setup() {
  // One shared shopper account: create it (409 = it already exists), then sign in.
  const cred = JSON.stringify({ user: 'blackfriday-shopper', password: __ENV.SHOPPER_PASSWORD || 'blackfriday-2026', name: 'Black Friday Shopper' });
  http.post(`${BASE}/api/auth/signup`, cred, JSON_HEADERS);
  const r = http.post(`${BASE}/api/auth/login`, cred, JSON_HEADERS);
  check(r, { 'login ok': (x) => x.status === 200 });
  return { token: r.json('token') };
}

export default function (data) {
  const [from, to] = ROUTES[Math.floor(Math.random() * ROUTES.length)];
  const date = new Date(Date.now() + 86400000 * (1 + Math.floor(Math.random() * 30))).toISOString().slice(0, 10);

  const res = http.get(`${BASE}/api/search/trains?from=${from}&to=${to}&date=${date}`, { tags: { name: 'search' } });
  check(res, { 'search 200': (r) => r.status === 200 });

  if (res.status === 200 && Math.random() < 0.15) {
    const results = res.json('results') || [];
    if (results.length) {
      const t = results[Math.floor(Math.random() * results.length)];
      const b = http.post(`${BASE}/api/booking/book`, JSON.stringify({
        train: t.train, from: t.from, to: t.to, date, class: t.classes[0].class, quota: 'GN',
        passengers: [{ name: 'Sale Shopper', age: 30, gender: 'F' }], payment: 'card', coupon: 'BLACKFRIDAY',
      }), { headers: { ...JSON_HEADERS.headers, Authorization: `Bearer ${data.token}` }, tags: { name: 'book' } });
      check(b, { 'book 201': (r) => r.status === 201 });
    }
  }
  if (Math.random() < 0.2) http.get(`${BASE}/api/search-indexer/suggest?q=del`, { tags: { name: 'suggest' } });
  sleep(Math.random() * 0.5 + 0.2);
}
