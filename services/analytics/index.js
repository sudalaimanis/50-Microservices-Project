// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// analytics — collects business events (bookings, cancellations) and serves live KPIs.
const kit = require('railshack-kit');
const s = kit.service('analytics');
const stats = { bookings: 0, cancellations: 0, revenue: 0, refunds: 0, byClass: {}, byRoute: {}, byTrain: {} };
s.post('/api/analytics/event', (c) => {
  const e = c.body || {};
  if (e.type === 'booking') {
    stats.bookings++; stats.revenue += e.amount || 0;
    stats.byClass[e.class] = (stats.byClass[e.class] || 0) + 1;
    stats.byRoute[e.route] = (stats.byRoute[e.route] || 0) + 1;
    stats.byTrain[e.train] = (stats.byTrain[e.train] || 0) + 1;
  } else if (e.type === 'cancellation') { stats.cancellations++; stats.refunds += e.amount || 0; }
  c.json(202, { accepted: true });
});
s.get('/api/analytics/summary', (c) => c.json(200, { ...stats, revenue: Math.round(stats.revenue), pod: kit.pod(),
  note: 'Counters are per replica; production would ship events to a stream like Kafka.' }));
s.run();
