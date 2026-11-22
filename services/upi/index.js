// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// upi — mock UPI collect request (the "approve on your phone" step).
const kit = require('railshack-kit');
const s = kit.service('upi');
s.post('/api/upi/collect', async (c) => {
  const { user, amount, txn } = c.body || {};
  if (!user || !(amount > 0)) return c.error(400, 'user and amount are required');
  await new Promise((r) => setTimeout(r, 150)); // simulated bank round trip
  c.json(200, { status: 'SUCCESS', vpa: `${user}@shackpay`, amount, txn, bankRef: `UPI${Date.now()}` });
});
s.run();
