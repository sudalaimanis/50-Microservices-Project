// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// user-profile — traveller profile (name, tier, saved travellers). Any username works in the demo.
const kit = require('railshack-kit');
const s = kit.service('user-profile');
const tiers = ['Silver', 'Gold', 'Platinum'];
s.get('/api/user-profile/:user', (c) => {
  const u = c.params.user;
  const n = [...u].reduce((a, ch) => a + ch.charCodeAt(0), 0);
  c.json(200, { user: u, displayName: u.charAt(0).toUpperCase() + u.slice(1), tier: tiers[n % 3],
    memberSince: 2018 + (n % 7), homeStation: ['NDLS', 'BSB', 'BCT', 'SBC', 'HWH'][n % 5], loyaltyPoints: (n * 37) % 9000 });
});
s.run();
