// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// waitlist — WL numbers per train/date/class, and promotion when someone cancels.
const kit = require('railshack-kit');
const s = kit.service('waitlist');
const queues = new Map();
const key = (b) => `${b.train}|${b.date}|${b.class}`;
s.post('/api/waitlist/add', (c) => {
  const k = key(c.body);
  const q = queues.get(k) || [];
  q.push(c.body.name);
  queues.set(k, q);
  const pos = q.length + 7;
  c.json(201, { name: c.body.name, status: `WL${pos}`, position: pos, label: `WL/${pos}`,
    confirmChance: pos < 15 ? 'High' : pos < 40 ? 'Medium' : 'Low' });
});
s.post('/api/waitlist/promote', (c) => {
  const q = queues.get(key(c.body)) || [];
  const promoted = q.splice(0, Number(c.body.seats || 1));
  c.json(200, { promoted, remaining: q.length });
});
s.get('/api/waitlist/length', (c) => c.json(200, { length: (queues.get(key(c.query)) || []).length }));
s.run();
