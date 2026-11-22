// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// food-ordering — order meals to your seat at upcoming stations (checks the PNR first).
const kit = require('railshack-kit');
const s = kit.service('food-ordering');
const menus = {
  default: [
    { item: 'Veg Thali', price: 180, veg: true }, { item: 'Paneer Biryani', price: 220, veg: true },
    { item: 'Chicken Biryani', price: 260, veg: false }, { item: 'Masala Dosa', price: 140, veg: true },
    { item: 'Chai + Samosa', price: 60, veg: true },
  ],
  BSB: [{ item: 'Banarasi Kachori Sabzi', price: 90, veg: true }, { item: 'Malaiyo', price: 80, veg: true }, { item: 'Banarasi Paan', price: 40, veg: true }],
  ASR: [{ item: 'Amritsari Kulcha', price: 150, veg: true }, { item: 'Lassi', price: 70, veg: true }],
};
s.get('/api/food-ordering/menu', (c) => c.json(200, { station: c.query.station, items: menus[c.query.station] || menus.default }));
s.post('/api/food-ordering/order', async (c) => {
  const { pnr, station, items = [] } = c.body || {};
  const b = await c.call('pnr', 'GET', `/api/pnr/${pnr}`);
  const total = items.reduce((a, it) => a + (it.price || 0) * (it.qty || 1), 0);
  c.json(201, { orderId: `FD${Date.now()}`, pnr, station, seat: b.passengers && b.passengers[0] && b.passengers[0].label, total, eta: 'Delivered at your seat on arrival' });
});
s.run();
