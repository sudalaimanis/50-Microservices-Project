// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// rac — Reservation Against Cancellation: two passengers share a side-lower berth.
const kit = require('railshack-kit');
const s = kit.service('rac');
const counters = new Map();
s.post('/api/rac/assign', (c) => {
  const k = `${c.body.train}|${c.body.date}|${c.body.class}`;
  const n = (counters.get(k) || 0) + 1;
  counters.set(k, n);
  if (n > 40 || c.body.class === '1A') return c.json(200, { name: c.body.name, status: 'NONE' });
  const berth = 7 + (Math.ceil(n / 2) - 1) * 8;
  c.json(201, { name: c.body.name, status: 'RAC', racNo: n, coach: c.body.class === 'SL' ? 'S4' : 'B2', berth,
    berthType: 'SL', label: `RAC ${n}/${berth}/SL` });
});
s.run();
