// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

// frontend — serves the RailShack web app and forwards /api/* to the api-gateway.
const fs = require('fs');
const path = require('path');
const kit = require('railshack-kit');
const s = kit.service('frontend');

const servicesFile = [process.env.SERVICES_FILE, path.join(__dirname, 'services.json'), path.join(__dirname, '../../services.json')]
  .find((f) => f && fs.existsSync(f));
const services = JSON.parse(fs.readFileSync(servicesFile, 'utf8'));

s.get('/services.json', (c) => c.json(200, services));
s.get('/runtime.json', (c) => c.json(200, { cluster: process.env.CLUSTER_NAME || 'local', environment: process.env.APP_ENV || 'dev',
  version: process.env.SERVICE_VERSION || '1.0.0', pod: kit.pod() }));
s.proxy('/api/', 'api-gateway');
s.static(path.join(__dirname, 'public'));
s.run();
