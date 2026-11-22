// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// email — outbound email delivery (mock provider). Keeps the last 50 messages for the admin view.
const kit = require('railshack-kit');
const s = kit.service('email');
const outbox = [];
s.post('/api/email/send', (c) => {
  const m = { id: `EMAIL-${Date.now()}`, ...c.body, sentAt: new Date().toISOString() };
  outbox.unshift(m); outbox.length = Math.min(outbox.length, 50);
  c.json(202, { queued: true, id: m.id });
});
s.get('/api/email/outbox', (c) => c.json(200, { count: outbox.length, messages: outbox.slice(0, 10) }));
s.run();
