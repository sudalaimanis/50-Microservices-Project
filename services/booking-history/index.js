// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// booking-history — "My Trips". Only the signed-in user may read their own trips.
// PostgreSQL (table trips) is the source of truth; Redis keeps a 5-minute copy of each user's list
// (cache-aside). A new trip is written to PostgreSQL first, then the user's cached list is deleted.
// Without DATABASE_URL (local runs) trips are kept in memory.
const kit = require('railshack-kit');
const s = kit.service('booking-history');
const db = kit.pg(process.env.DATABASE_URL);
const cache = kit.redis(process.env.REDIS_ADDR);
const TTL = 300;
const mem = new Map();

let schema = null;
function ensure() {
  if (!schema) {
    schema = db.query(`CREATE TABLE IF NOT EXISTS trips (
        id       BIGSERIAL PRIMARY KEY,
        username TEXT NOT NULL,
        pnr      TEXT NOT NULL,
        data     JSONB NOT NULL,
        saved_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
      .then(() => db.query('CREATE INDEX IF NOT EXISTS trips_user_time ON trips (username, saved_at DESC)'))
      .catch((e) => { schema = null; throw e; });
  }
  return schema;
}

s.post('/api/booking-history/add', async (c) => {
  const trip = { ...c.body, savedAt: new Date().toISOString() };
  if (!trip.user) return c.error(400, 'user is required');
  if (db) {
    await ensure();
    await db.query('INSERT INTO trips (username, pnr, data) VALUES ($1, $2, $3)', [trip.user, trip.pnr || '', JSON.stringify(trip)]);
    if (cache) await cache.del(`trips:${trip.user}`); // the list changed: drop the old copy
  } else {
    const l = mem.get(trip.user) || []; l.unshift(trip); mem.set(trip.user, l.slice(0, 50));
  }
  c.json(201, { saved: true });
});

s.get('/api/booking-history/:user', async (c) => {
  const user = await c.whoami(); // 401 unless signed in
  if (user !== c.params.user) return c.error(403, 'you can only see your own trips');
  if (!db) return c.json(200, { user, trips: mem.get(user) || [], servedFrom: 'memory' });
  const hit = cache && await cache.get(`trips:${user}`);
  if (hit) return c.json(200, { user, trips: JSON.parse(hit), servedFrom: 'redis' }); // cache hit
  await ensure();
  const r = await db.query('SELECT data FROM trips WHERE username = $1 ORDER BY saved_at DESC LIMIT 50', [user]);
  const trips = r.rows.map((row) => JSON.parse(row.data));
  if (cache) await cache.set(`trips:${user}`, JSON.stringify(trips), TTL); // cache miss: keep a copy
  c.json(200, { user, trips, servedFrom: 'postgres' });
});
s.run();
