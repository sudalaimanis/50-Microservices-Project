// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// offers — coupon codes and payment-method offers.
const kit = require('railshack-kit');
const s = kit.service('offers');
const coupons = {
  BLACKFRIDAY: { pct: 20, max: 500, text: 'Black Friday: 20% off (max ₹500)' },
  SHACK50: { flat: 50, text: 'Flat ₹50 off' },
  DEVOPS16: { pct: 10, max: 300, text: 'DevOps Shack Batch 16 special: 10% off' },
};
s.get('/api/offers', (c) => c.json(200, Object.entries(coupons).map(([code, o]) => ({ code, text: o.text }))));
s.post('/api/offers/apply', (c) => {
  const { coupon, amount = 0, payment } = c.body || {};
  let discount = 0; const applied = [];
  const o = coupon && coupons[String(coupon).toUpperCase()];
  if (o) { discount += o.flat || Math.min(o.max, (amount * o.pct) / 100); applied.push(o.text); }
  if (payment === 'upi') { discount += 15; applied.push('₹15 UPI cashback'); }
  c.json(200, { discount: Math.round(discount * 100) / 100, applied, validCoupon: !!o || !coupon });
});
s.run();
