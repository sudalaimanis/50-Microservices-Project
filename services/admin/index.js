// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// admin — operations dashboard API. Aggregates analytics, audit, settlements and platform health.
const kit = require('railshack-kit');
const s = kit.service('admin');
s.get('/api/admin/dashboard', async (c) => {
  const safe = (p) => p.catch((e) => ({ error: e.message }));
  const [analytics, audit, settlements, platform, feedback] = await Promise.all([
    safe(c.call('analytics', 'GET', '/api/analytics/summary')),
    safe(c.call('audit-log', 'GET', '/api/audit-log/recent')),
    safe(c.call('settlements', 'GET', '/api/settlements/summary')),
    safe(c.call('api-gateway', 'GET', '/api/platform/status')),
    safe(c.call('feedback', 'GET', '/api/feedback/summary')),
  ]);
  c.json(200, { analytics, audit, settlements, feedback,
    platform: platform.services ? { total: platform.total, up: platform.up } : platform });
});
s.run();
