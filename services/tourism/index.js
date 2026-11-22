// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// tourism — curated rail holiday packages by destination.
const kit = require('railshack-kit');
const s = kit.service('tourism');
const packages = [
  { id: 'T1', title: 'Kashi Vishwanath & Ganga Aarti', city: 'Varanasi', station: 'BSB', nights: 2, price: 7499 },
  { id: 'T2', title: 'Golden Temple & Wagah Border', city: 'Amritsar', station: 'ASR', nights: 2, price: 6999 },
  { id: 'T3', title: 'Pink City Heritage Walk', city: 'Jaipur', station: 'JP', nights: 3, price: 9999 },
  { id: 'T4', title: 'Taj Mahal Sunrise Express', city: 'Agra', station: 'AGC', nights: 1, price: 4499 },
  { id: 'T5', title: 'Mumbai Food Trail', city: 'Mumbai', station: 'BCT', nights: 2, price: 8999 },
];
s.get('/api/tourism/packages', (c) => c.json(200, c.query.station ? packages.filter((p) => p.station === c.query.station) : packages));
s.run();
