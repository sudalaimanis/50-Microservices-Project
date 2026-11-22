// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// cancellation — works out cancellation charges, triggers the refund, frees the seat
// for the waitlist and tells the passenger. Only the person who booked the ticket may cancel it.
const kit = require('railshack-kit');
const s = kit.service('cancellation');
const flat = { '1A': 240, '2A': 200, '3A': 180, CC: 180, EC: 240, SL: 120, '2S': 60 };
s.post('/api/cancellation/cancel', async (c) => {
  const pnr = c.body && c.body.pnr;
  if (!pnr) return c.error(400, 'pnr is required');
  const user = await c.whoami(); // 401 unless signed in
  const b = await c.call('pnr', 'GET', `/api/pnr/${pnr}`);
  if (b.user !== user) return c.error(403, 'you can only cancel your own tickets');
  if (b.state === 'CANCELLED') return c.error(409, 'this PNR is already cancelled');
  const pax = (b.passengers || []).length || 1;
  const charges = Math.min(b.total, (flat[b.class] || 120) * pax);
  const refundAmount = Math.round((b.total - charges) * 100) / 100;
  const refund = await c.call('refunds', 'POST', '/api/refunds/initiate',
    { user: b.user, txn: b.payment && b.payment.txn, amount: refundAmount, method: b.payment && b.payment.method });
  await c.call('pnr', 'POST', `/api/pnr/${pnr}/cancel`);
  await c.fire('waitlist', 'POST', '/api/waitlist/promote', { train: b.train, date: b.date, class: b.class, seats: pax });
  await c.fire('notifications', 'POST', '/api/notifications/send', { user: b.user, template: 'cancelled', data: { pnr } });
  await c.fire('audit-log', 'POST', '/api/audit-log/record', { actor: b.user, action: 'CANCEL', resource: `pnr/${pnr}` });
  await c.fire('analytics', 'POST', '/api/analytics/event', { type: 'cancellation', train: b.train, amount: refundAmount });
  c.json(200, { pnr, cancelled: true, charges, refund });
});
s.run();
