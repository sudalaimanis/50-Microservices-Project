// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// feedback — star ratings and comments after a journey.
const kit = require('railshack-kit');
const s = kit.service('feedback');
const all = [];
s.post('/api/feedback', (c) => {
  const r = Number(c.body && c.body.rating);
  if (!(r >= 1 && r <= 5)) return c.error(400, 'rating must be 1-5');
  all.unshift({ ...c.body, at: new Date().toISOString() });
  c.json(201, { thanks: true });
});
s.get('/api/feedback/summary', (c) => {
  const avg = all.length ? all.reduce((a, f) => a + Number(f.rating), 0) / all.length : 0;
  c.json(200, { count: all.length, average: Math.round(avg * 10) / 10, latest: all.slice(0, 5) });
});
s.run();
