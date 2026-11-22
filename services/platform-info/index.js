// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// platform-info — expected platform and coach position for a train at a station.
const kit = require('railshack-kit');
const s = kit.service('platform-info');
s.get('/api/platform-info/:station', async (c) => {
  const st = await c.call('stations', 'GET', `/api/stations/${c.params.station}`);
  const n = [...(c.query.train || '') + c.params.station].reduce((a, ch) => a + ch.charCodeAt(0), 0);
  c.json(200, { station: st.code, stationName: st.name, train: c.query.train, platform: (n % st.platforms) + 1,
    coachPosition: 'ENG-H1-A1-A2-B1-B2-B3-B4-B5-B6-PC-S1-S2-S3-GS-SLR' });
});
s.run();
