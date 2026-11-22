// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// preferences — notification channels, meal choice and language per user.
const kit = require('railshack-kit');
const s = kit.service('preferences');
const store = new Map();
s.get('/api/preferences/:user', (c) => {
  c.json(200, store.get(c.params.user) || { user: c.params.user, channels: ['email', 'sms'], meal: 'veg', language: 'en' });
});
s.put('/api/preferences/:user', (c) => {
  store.set(c.params.user, { user: c.params.user, ...c.body });
  c.json(200, store.get(c.params.user));
});
s.run();
