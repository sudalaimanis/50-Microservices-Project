// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// passengers — validates traveller details and applies Indian Railways rules (age, berth preference, concessions).
const kit = require('railshack-kit');
const s = kit.service('passengers');
s.post('/api/passengers/validate', (c) => {
  const list = (c.body && c.body.passengers) || [];
  if (!list.length || list.length > 6) return c.error(400, 'between 1 and 6 passengers allowed per PNR');
  const out = [];
  for (const [i, p] of list.entries()) {
    const name = String(p.name || '').trim();
    const age = Number(p.age);
    if (name.length < 2) return c.error(400, `passenger ${i + 1}: name is required`);
    if (!(age >= 1 && age <= 120)) return c.error(400, `passenger ${i + 1}: age must be 1-120`);
    out.push({ name, age, gender: p.gender || 'NA', berthPreference: p.berthPreference || (age >= 60 ? 'LB' : 'ANY'),
      seniorCitizen: age >= 60, child: age < 12 });
  }
  c.json(200, { passengers: out, count: out.length });
});
s.run();
