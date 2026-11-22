// RailShack · 50 Microservices Project by DevOps Shack
// Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
// SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.

/* RailShack frontend · DevOps Shack
 * Vanilla JS, no build step. Every API call goes through the api-gateway; the
 * X-Trace response header lists every microservice the request touched, and we
 * animate those services on the mesh.
 */
console.log('%c RailShack %c 50 Microservices Project by DevOps Shack ',
  'background:#00ffa3;color:#001510;font-weight:700;padding:4px 8px;border-radius:4px 0 0 4px',
  'background:#05060a;color:#00ffa3;padding:4px 8px;border:1px solid #00ffa3;border-radius:0 4px 4px 0');
console.log('Built by DevOps Shack · https://devopsshack.com · https://www.youtube.com/@devopsshack');
const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const state = { services: [], byName: {}, token: localStorage.getItem('rs_token'), user: localStorage.getItem('rs_user'), quota: 'GN', stations: [] };
const LANG_COLOR = { go: 'var(--go)', node: 'var(--node)', python: 'var(--python)', java: 'var(--java)' };
const inr = (n) => '₹' + Number(n || 0).toLocaleString('en-IN', { maximumFractionDigits: 2 });
const servedBadge = (src) => src === 'redis' ? '<span class="served hit">⚡ served from Redis cache</span>'
  : src === 'postgres' ? '<span class="served miss">🐘 read from PostgreSQL · now cached in Redis</span>' : '';
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

/* ---------------- API + trace ---------------- */
async function api(path, { method = 'GET', body, title, quiet } = {}) {
  const headers = { 'Content-Type': 'application/json', 'X-Request-Id': crypto.randomUUID?.() || String(Date.now()) };
  if (state.token) headers.Authorization = 'Bearer ' + state.token;
  const res = await fetch(path, { method, headers,
    body: body ? JSON.stringify(body) : undefined });
  const trace = (res.headers.get('X-Trace') || '').split(',').filter(Boolean).filter((s) => s !== 'frontend');
  if (trace.length && !quiet) showTrace(title || `${method} ${path.split('?')[0]}`, ['frontend', ...trace]);
  let data = null;
  try { data = await res.json(); } catch { data = null; }
  if (!res.ok) { const e = new Error((data && data.error) || `HTTP ${res.status}`); e.status = res.status; throw e; }
  return data;
}

let traceTimer;
function showTrace(title, list) {
  const uniq = [...new Set(list)];
  $('#traceTitle').textContent = title;
  $('#traceCount').textContent = `${uniq.length} microservices`;
  const chips = $('#traceChips');
  chips.innerHTML = '';
  uniq.forEach((name, i) => {
    if (i) chips.insertAdjacentHTML('beforeend', `<span class="chip arrow" style="animation-delay:${i * 70}ms">→</span>`);
    const lang = state.byName[name]?.lang;
    chips.insertAdjacentHTML('beforeend', `<span class="chip" style="--c:${LANG_COLOR[lang] || 'var(--cyan)'};animation-delay:${i * 70}ms">${esc(name)}</span>`);
  });
  $$('#miniMesh i').forEach((el) => el.classList.remove('on'));
  uniq.forEach((name, i) => setTimeout(() => {
    $(`#miniMesh i[data-s="${name}"]`)?.classList.add('on');
    const tile = $(`.tile[data-s="${name}"]`);
    if (tile) { tile.classList.remove('hit'); void tile.offsetWidth; tile.classList.add('hit'); }
  }, i * 90));
  $('#traceDock').classList.add('open');
  clearTimeout(traceTimer);
  traceTimer = setTimeout(() => $('#traceDock').classList.remove('open'), 9000);
}
$('#traceClose').onclick = () => $('#traceDock').classList.remove('open');

function toast(msg, err = false) {
  const t = $('#toast');
  t.textContent = msg;
  t.className = 'toast show' + (err ? ' err' : '');
  clearTimeout(t._t);
  t._t = setTimeout(() => (t.className = 'toast'), 3800);
}

/* ---------------- theme ---------------- */
function applyTheme(t) {
  if (t === 'dark') document.documentElement.dataset.theme = 'dark'; else delete document.documentElement.dataset.theme;
  $('#themeToggle').textContent = t === 'dark' ? '☀' : '☾';
  try { localStorage.setItem('rs_theme', t); } catch { /* storage unavailable */ }
}
applyTheme(document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light');
$('#themeToggle').onclick = () => applyTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark');

/* ---------------- routing ---------------- */
function go(view) {
  if (!$('#view-' + view)) view = 'search';
  $('#drawer').classList.remove('open');
  $$('.view').forEach((v) => v.classList.toggle('active', v.id === 'view-' + view));
  $$('.links a').forEach((a) => a.classList.toggle('active', a.dataset.view === view));
  if (location.hash !== '#' + view) history.replaceState(null, '', '#' + view);
  ({ trips: loadTrips, admin: loadAdmin, extras: loadExtras, mesh: pollHealth }[view] || (() => {}))();
  window.scrollTo({ top: 0, behavior: 'smooth' });
}
document.addEventListener('click', (e) => {
  const v = e.target.closest('[data-view]');
  if (v) { e.preventDefault(); go(v.dataset.view); }
});

window.addEventListener('hashchange', () => go(location.hash.slice(1)));

/* ---------------- auth ---------------- */
// Open to everyone: search, live status, PNR, extras. Need an account: booking, My Trips, cancelling.
let authMode = 'login', authWaiters = [];
function setSession(r) {
  state.token = r ? r.token : null; state.user = r ? r.user : null;
  try {
    if (r) { localStorage.setItem('rs_token', r.token); localStorage.setItem('rs_user', r.user); }
    else { localStorage.removeItem('rs_token'); localStorage.removeItem('rs_user'); }
  } catch { /* storage unavailable */ }
  $('#userChip').textContent = state.user || 'Sign in';
  $('#signOut').classList.toggle('hidden', !state.user);
}
function setAuthMode(m) {
  authMode = m;
  $$('.auth-tabs button').forEach((b) => b.classList.toggle('on', b.dataset.mode === m));
  $$('.signup-only').forEach((el) => el.classList.toggle('hidden', m !== 'signup'));
  $$('.login-only').forEach((el) => el.classList.toggle('hidden', m !== 'login'));
  $('#loginSubmit').textContent = m === 'signup' ? 'Create account' : 'Sign in';
  $('#loginPass').autocomplete = m === 'signup' ? 'new-password' : 'current-password';
  $('#loginError').classList.add('hidden');
}
function openAuth(reason = '', mode = 'login') {
  setAuthMode(mode);
  $('#loginReason').textContent = reason;
  $('#loginReason').classList.toggle('hidden', !reason);
  $('#loginModal').classList.add('open');
  setTimeout(() => (mode === 'signup' ? $('#loginName') : $('#loginUser')).focus(), 50);
}
function closeAuth(ok) {
  $('#loginModal').classList.remove('open');
  const w = authWaiters; authWaiters = [];
  w.forEach(([res, rej]) => (ok ? res(state.user) : rej(new Error('sign-in cancelled'))));
}
// requireLogin resolves once the user is signed in (straight away if they already are).
function requireLogin(reason) {
  if (state.token) return Promise.resolve(state.user);
  return new Promise((res, rej) => { authWaiters.push([res, rej]); openAuth(reason); });
}
// Runs fn; if the server says the session is gone (401), asks the user to sign in and tries once more.
async function withLogin(reason, fn) {
  await requireLogin(reason);
  try { return await fn(); } catch (err) {
    if (err.status !== 401) throw err;
    setSession(null);
    await requireLogin('Your session has expired. Please sign in again.');
    return fn();
  }
}
$$('.auth-tabs button').forEach((b) => (b.onclick = () => setAuthMode(b.dataset.mode)));
$$('[data-switch]').forEach((a) => (a.onclick = (e) => { e.preventDefault(); setAuthMode(a.dataset.switch); }));
$('#userChip').onclick = () => { if (state.user) go('trips'); else openAuth(); };
$('#signOut').onclick = () => { setSession(null); toast('Signed out'); if ($('#view-trips').classList.contains('active')) loadTrips(); };
$('#loginClose').onclick = () => closeAuth(false);
$('#loginModal').onclick = (e) => { if (e.target.id === 'loginModal') closeAuth(false); };
$('#loginForm').onsubmit = async (e) => {
  e.preventDefault();
  const user = $('#loginUser').value.trim(), password = $('#loginPass').value, name = $('#loginName').value.trim();
  const err = $('#loginError'), btn = $('#loginSubmit');
  err.classList.add('hidden'); btn.disabled = true;
  try {
    const r = authMode === 'signup'
      ? await api('/api/auth/signup', { method: 'POST', body: { user, password, name }, title: 'Create account' })
      : await api('/api/auth/login', { method: 'POST', body: { user, password }, title: 'Sign in' });
    setSession(r);
    $('#loginPass').value = '';
    toast(authMode === 'signup' ? `Welcome aboard, ${r.name || r.user}!` : `Welcome back, ${r.name || r.user}!`);
    closeAuth(true);
  } catch (ex) {
    err.textContent = ex.message; err.classList.remove('hidden');
  } finally { btn.disabled = false; }
};
async function restoreSession() {
  setSession(state.token ? { token: state.token, user: state.user } : null);
  if (!state.token) return;
  try { await api('/api/auth/verify', { quiet: true }); } catch { setSession(null); }
}

/* ---------------- mesh ---------------- */
function buildMesh() {
  const domains = {};
  state.services.forEach((s) => (domains[s.domain] = domains[s.domain] || []).push(s));
  $('#mesh').innerHTML = Object.entries(domains).map(([d, list]) => `
    <div class="domain glass"><h4>${esc(d)} <span>${list.length}</span></h4><div class="tiles">
      ${list.map((s) => `<div class="tile" data-s="${s.name}"><div class="nm"><i class="lang ${s.lang}"></i>${esc(s.name)}</div><div class="pd">waiting…</div></div>`).join('')}
    </div></div>`).join('');
  $('#miniMesh').innerHTML = state.services.map((s) => `<i data-s="${s.name}" title="${s.name}"></i>`).join('');
}
async function pollHealth() {
  try {
    const st = await api('/api/platform/status', { quiet: true });
    const badge = $('#healthBadge');
    $('#healthText').textContent = `${st.up}/${st.total} services up`;
    badge.classList.toggle('bad', st.up < st.total);
    $('#meshSummary').textContent = `${st.up}/${st.total} healthy · swept by gateway pod ${st.gatewayPod}`;
    st.services.forEach((r) => {
      const t = $(`.tile[data-s="${r.service}"]`);
      if (!t) return;
      t.classList.toggle('up', r.up); t.classList.toggle('down', !r.up);
      $('.pd', t).textContent = r.up ? `${r.pod || ''} · ${r.latencyMs}ms` : 'unreachable';
    });
  } catch {
    $('#healthText').textContent = 'gateway unreachable';
    $('#healthBadge').classList.add('bad');
  }
}

/* ---------------- station autocomplete ---------------- */
function stationInput(inputId, boxId) {
  const input = $(inputId), box = $(boxId);
  let t, items = [], sel = -1;
  const render = () => {
    box.innerHTML = items.map((s, i) => `<div class="${i === sel ? 'sel' : ''}" data-i="${i}"><span>${esc(s.name)}<small style="color:var(--muted)"> · ${esc(s.city)}</small></span><b>${s.code}</b></div>`).join('');
    box.classList.toggle('open', items.length > 0);
  };
  const pick = (s) => { input.value = `${s.name} (${s.code})`; input.dataset.code = s.code; box.classList.remove('open'); };
  input.addEventListener('input', () => {
    input.dataset.code = '';
    clearTimeout(t);
    t = setTimeout(async () => {
      try { items = await api('/api/search-indexer/suggest?q=' + encodeURIComponent(input.value), { quiet: true }); sel = -1; render(); } catch { /* ignore */ }
    }, 150);
  });
  input.addEventListener('focus', () => input.dispatchEvent(new Event('input')));
  input.addEventListener('keydown', (e) => {
    if (!box.classList.contains('open')) return;
    if (e.key === 'ArrowDown') { sel = Math.min(sel + 1, items.length - 1); render(); e.preventDefault(); }
    if (e.key === 'ArrowUp') { sel = Math.max(sel - 1, 0); render(); e.preventDefault(); }
    if (e.key === 'Enter' && sel >= 0) { pick(items[sel]); e.preventDefault(); }
  });
  box.addEventListener('mousedown', (e) => { const d = e.target.closest('[data-i]'); if (d) pick(items[d.dataset.i]); });
  input.addEventListener('blur', () => setTimeout(() => box.classList.remove('open'), 150));
  return { set: (s) => pick(s) };
}
const codeOf = (input) => input.dataset.code || (input.value.match(/\(([A-Z]+)\)/) || [])[1] || input.value.trim().toUpperCase();

/* ---------------- search ---------------- */
let fromAC, toAC, lastSearch;
$('#swap').onclick = () => {
  const f = $('#from'), t = $('#to');
  [f.value, t.value] = [t.value, f.value];
  [f.dataset.code, t.dataset.code] = [t.dataset.code, f.dataset.code];
};
$$('#quotaSeg button').forEach((b) => (b.onclick = () => {
  $$('#quotaSeg button').forEach((x) => x.classList.toggle('on', x === b));
  state.quota = b.dataset.q;
}));
$$('.quick [data-route]').forEach((b) => (b.onclick = () => {
  const [f, t] = b.dataset.route.split(',');
  const byCode = Object.fromEntries(state.stations.map((s) => [s.code, s]));
  if (byCode[f]) fromAC.set(byCode[f]);
  if (byCode[t]) toAC.set(byCode[t]);
  $('#searchForm').requestSubmit();
}));

$('#searchForm').onsubmit = async (e) => {
  e.preventDefault();
  const from = codeOf($('#from')), to = codeOf($('#to')), date = $('#date').value;
  if (from === to) return toast('From and To cannot be the same', true);
  const box = $('#results');
  box.innerHTML = '<div class="skeleton"></div><div class="skeleton"></div><div class="skeleton"></div>';
  try {
    const r = await api(`/api/search/trains?from=${from}&to=${to}&date=${date}`, { title: `Search ${from} → ${to}` });
    lastSearch = r;
    renderResults(r);
  } catch (err) {
    box.innerHTML = `<div class="empty glass">⚠️ ${esc(err.message)}</div>`;
  }
};

function renderResults(r) {
  const box = $('#results');
  if (!r.results.length) {
    box.innerHTML = `<div class="empty glass">No direct trains from <b>${esc(r.from)}</b> to <b>${esc(r.to)}</b>. Try Delhi → Varanasi or Mumbai → Delhi.</div>`;
    return;
  }
  box.innerHTML = `<div class="results-head"><h2>${esc(r.fromName || r.from)} → ${esc(r.toName || r.to)}</h2>
    <span>${r.count} trains · ${new Date(r.date).toDateString()} · served by ${esc(r.servedBy)}</span></div>` +
    r.results.map((t, i) => `
    <article class="tcard glass" style="animation-delay:${i * 90}ms">
      <div>
        <div class="tno">#${t.train}</div>
        <div class="tname">${esc(t.name)}<span class="badge ${t.type.split(' ')[0]}">${esc(t.type)}</span></div>
        <div class="journey">
          <div><div class="t">${t.departs}</div><small>${t.from}</small></div>
          <div class="line"><span>${t.duration} · ${t.km} km</span><i></i></div>
          <div style="text-align:right"><div class="t">${t.arrives}</div><small>${t.to}${t.dayDiff ? ` · +${t.dayDiff}d` : ''}</small></div>
        </div>
        <div class="meta-row"><span>${t.runsOnDate ? '✅ Runs on this date' : '⚠️ Does not run on this date'}</span>${t.pantry ? '<span>🍽 Pantry</span>' : ''}</div>
      </div>
      <div class="classes">
        ${t.classes.map((c) => `<button class="cls" data-train="${t.train}" data-class="${c.class}">
          <b>${c.class}</b><span class="st ${c.status}">${esc(c.label)}</span><span class="fare">${inr(c.fare)}</span></button>`).join('')}
      </div>
    </article>`).join('');
  $$('.cls', box).forEach((b) => (b.onclick = () => openBooking(b.dataset.train, b.dataset.class)));
}

/* ---------------- booking drawer ---------------- */
let draft;
async function openBooking(trainNo, cls) {
  try { await requireLogin('Sign in to book this ticket. Search and live status stay open to everyone.'); } catch { return; }
  const t = lastSearch.results.find((x) => x.train === trainNo);
  const c = t.classes.find((x) => x.class === cls);
  draft = { t, c, passengers: [{ name: state.user ? state.user[0].toUpperCase() + state.user.slice(1) : '', age: 28, gender: 'M' }], payment: 'upi', coupon: '', insurance: true };
  renderDrawer();
  $('#drawer').classList.add('open');
}
function renderDrawer() {
  const { t, c } = draft;
  $('#drawerBody').innerHTML = `
    <h3>${esc(t.name)} <span class="tno">#${t.train}</span></h3>
    <div class="sub">${t.from} ${t.departs} → ${t.to} ${t.arrives} · ${lastSearch.date} · Class <b>${c.class}</b> · <span class="st ${c.status}">${esc(c.label)}</span> · Quota ${state.quota}</div>
    <label class="field"><span style="font:600 11px var(--mono);letter-spacing:.12em;color:var(--muted)">PASSENGERS</span></label>
    <div id="paxList">${draft.passengers.map((p, i) => `
      <div class="pax"><input data-i="${i}" data-k="name" placeholder="Full name" value="${esc(p.name)}" />
      <input data-i="${i}" data-k="age" type="number" min="1" max="120" value="${p.age}" />
      <select data-i="${i}" data-k="gender"><option ${p.gender === 'M' ? 'selected' : ''} value="M">Male</option><option ${p.gender === 'F' ? 'selected' : ''} value="F">Female</option><option ${p.gender === 'O' ? 'selected' : ''} value="O">Other</option></select>
      <button data-rm="${i}" title="Remove">✕</button></div>`).join('')}</div>
    <button class="btn" id="addPax" ${draft.passengers.length >= 6 ? 'disabled' : ''}>+ Add passenger</button>
    <div class="row"><input id="coupon" placeholder="Coupon (try BLACKFRIDAY)" value="${esc(draft.coupon)}" style="flex:1" />
      <label class="toggle"><input type="checkbox" id="ins" ${draft.insurance ? 'checked' : ''}/> Travel insurance (₹0.45)</label></div>
    <div class="pay">${['upi', 'wallet', 'card'].map((m) => `<button data-pay="${m}" class="${draft.payment === m ? 'on' : ''}">${{ upi: '⚡ UPI', wallet: '👛 Wallet', card: '💳 Card' }[m]}</button>`).join('')}</div>
    <div class="summary"><div><span>Fare / passenger (from search)</span><span>${inr(c.fare)}</span></div>
      <div><span>Passengers</span><span>× ${draft.passengers.length}</span></div>
      <div class="tot"><span>Estimated total</span><span>${inr(c.fare * draft.passengers.length)}</span></div>
      <div style="color:var(--muted);font-size:11px">Final price is computed by <code>fares</code>, <code>offers</code> and <code>insurance</code> at checkout.</div></div>
    <div class="steps hidden" id="steps"></div>
    <button class="btn primary big" id="bookBtn" style="width:100%;margin-top:16px"><span>Pay & book</span><i class="shine"></i></button>
    <div id="ticketBox"></div>`;
  $$('#paxList input, #paxList select').forEach((el) => (el.oninput = () => {
    const p = draft.passengers[el.dataset.i];
    p[el.dataset.k] = el.dataset.k === 'age' ? Number(el.value) : el.value;
  }));
  $$('[data-rm]').forEach((b) => (b.onclick = () => { if (draft.passengers.length > 1) { draft.passengers.splice(b.dataset.rm, 1); renderDrawer(); } }));
  $('#addPax').onclick = () => { draft.passengers.push({ name: '', age: 30, gender: 'F' }); renderDrawer(); };
  $('#coupon').oninput = (e) => (draft.coupon = e.target.value);
  $('#ins').onchange = (e) => (draft.insurance = e.target.checked);
  $$('[data-pay]').forEach((b) => (b.onclick = () => { draft.payment = b.dataset.pay; $$('[data-pay]').forEach((x) => x.classList.toggle('on', x === b)); }));
  $('#bookBtn').onclick = book;
}
$('#drawerClose').onclick = () => $('#drawer').classList.remove('open');
$('#drawer').onclick = (e) => { if (e.target.id === 'drawer') e.currentTarget.classList.remove('open'); };

const STEPS = ['Verifying you (auth · user-profile · kyc)', 'Validating passengers', 'Pricing (fares · offers · insurance)',
  'Allocating seats (seat-allocation · rac · waitlist)', 'Taking payment (payment-gateway · upi/wallet)', 'Generating PNR, invoice & e-ticket', 'Notifying you (notifications · email · sms)'];

async function book() {
  const btn = $('#bookBtn');
  btn.disabled = true;
  const steps = $('#steps');
  steps.classList.remove('hidden');
  steps.innerHTML = STEPS.map((s) => `<div class="step"><i></i>${s}</div>`).join('');
  const els = $$('.step', steps);
  let k = 0;
  els[0].classList.add('doing');
  const tick = setInterval(() => {
    if (k < els.length - 1) { els[k].classList.replace('doing', 'done'); els[k].querySelector('i').textContent = '✓'; k++; els[k].classList.add('doing'); }
  }, 280);
  try {
    const { t, c } = draft;
    const b = await api('/api/booking/book', { method: 'POST', title: `Booking ${t.train} · ${c.class}`, body: {
      train: t.train, from: t.from, to: t.to, date: lastSearch.date, class: c.class, quota: state.quota,
      passengers: draft.passengers, payment: draft.payment, coupon: draft.coupon, insurance: draft.insurance } });
    clearInterval(tick);
    els.forEach((e) => { e.classList.remove('doing'); e.classList.add('done'); e.querySelector('i').textContent = '✓'; });
    btn.classList.add('hidden');
    $('#ticketBox').innerHTML = ticketHTML(b);
    drawCode($('#ticketBox canvas'), (b.ticket && b.ticket.qr) || b.pnr);
    confetti();
    toast(`Booked! PNR ${b.pnr}`);
  } catch (err) {
    clearInterval(tick);
    if (err.status === 401) { setSession(null); $('#drawer').classList.remove('open'); openAuth('Your session has expired. Please sign in again, then book.'); }
    els[k].classList.remove('doing');
    els[k].querySelector('i').textContent = '!';
    toast(err.message, true);
    btn.disabled = false;
  }
}

function ticketHTML(b) {
  const f = b.fare || {};
  return `<div class="ticket">
    <div style="display:flex;justify-content:space-between;align-items:center;gap:10px;flex-wrap:wrap">
      <div><div style="color:var(--muted);font:600 11px var(--mono);letter-spacing:.14em">PNR</div><div class="pnr">${b.pnr}</div></div>
      <span class="status ${b.status}">${b.status}</span></div>
    <div style="margin-top:8px">${esc(b.trainName)} · #${b.train} · ${b.from} → ${b.to} · ${b.date} · ${b.class}</div>
    <div class="cut"></div>
    <table>${(b.passengers || []).map((p, i) => `<tr><td>${i + 1}. ${esc(p.name)}</td><td>${esc(p.label || p.status)}</td></tr>`).join('')}
      <tr><td>Base fare × ${f.passengers}</td><td>${inr(f.baseFare * f.passengers)}</td></tr>
      ${f.gst ? `<tr><td>GST</td><td>${inr(f.gst * f.passengers)}</td></tr>` : ''}
      ${b.offer && b.offer.discount ? `<tr><td>Offers (${esc(b.offer.applied.join(', '))})</td><td>− ${inr(b.offer.discount)}</td></tr>` : ''}
      ${b.insurance ? `<tr><td>Insurance</td><td>${inr(b.insurance.premium)}</td></tr>` : ''}
      <tr><td><b>Total paid</b> · ${esc(b.payment.method.toUpperCase())} · ${esc(b.payment.txn)}</td><td><b>${inr(b.total)}</b></td></tr></table>
    <div class="cut"></div>
    <div class="bottom"><div style="font-size:12px;color:var(--muted)">${b.invoice && b.invoice.invoiceNo ? `Invoice ${esc(b.invoice.invoiceNo)}<br/>` : ''}Chart prepares 4h before departure<br/>Show this code to the TTE</div><canvas width="29" height="29"></canvas></div>
  </div>`;
}

// Draws a QR-style code from a hash of the ticket payload (signed by ticket-pdf).
function drawCode(canvas, text) {
  if (!canvas) return;
  const n = 29, ctx = canvas.getContext('2d');
  let h = 2166136261;
  for (const ch of text) { h ^= ch.charCodeAt(0); h = Math.imul(h, 16777619); }
  const rnd = () => ((h = Math.imul(h ^ (h >>> 15), 2246822507) ^ Math.imul(h ^ (h >>> 13), 3266489909)) >>> 0) / 4294967296;
  ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, n, n); ctx.fillStyle = '#05060a';
  for (let y = 0; y < n; y++) for (let x = 0; x < n; x++) if (rnd() > 0.52) ctx.fillRect(x, y, 1, 1);
  [[0, 0], [n - 7, 0], [0, n - 7]].forEach(([x, y]) => {
    ctx.fillStyle = '#05060a'; ctx.fillRect(x, y, 7, 7); ctx.fillStyle = '#fff'; ctx.fillRect(x + 1, y + 1, 5, 5); ctx.fillStyle = '#05060a'; ctx.fillRect(x + 2, y + 2, 3, 3);
  });
  canvas.style.imageRendering = 'pixelated';
}

/* ---------------- trips / pnr ---------------- */
async function loadTrips() {
  const box = $('#tripsList');
  if (!state.token) {
    box.innerHTML = `<div class="empty glass signin-card"><b>Sign in to see your trips</b><span class="muted">My Trips shows only your own bookings, so we need to know who you are.</span>
      <button class="btn primary" id="tripsSignIn">Sign in</button><a href="#" id="tripsSignUp">or create an account</a></div>`;
    $('#tripsSignIn').onclick = () => requireLogin('Sign in to see your trips.').then(loadTrips).catch(() => {});
    $('#tripsSignUp').onclick = (e) => { e.preventDefault(); authWaiters.push([loadTrips, () => {}]); openAuth('Create an account to start booking.', 'signup'); };
    return;
  }
  box.innerHTML = '<div class="skeleton"></div>';
  try {
    const r = await api('/api/booking-history/' + state.user, { title: 'My Trips' });
    $('#tripsServed').innerHTML = servedBadge(r.servedFrom);
    box.innerHTML = r.trips.length ? r.trips.map((t, i) => `<div class="card glass" style="animation-delay:${i * 60}ms">
      <div class="tno">PNR ${t.pnr}</div><h4>${esc(t.trainName)}</h4>
      <div class="muted">${t.from} → ${t.to} · ${t.date} · ${t.class} · ${t.status}</div>
      <div class="price">${inr(t.total)}</div>
      <button class="btn" style="margin-top:10px" data-pnr="${t.pnr}">View PNR</button></div>`).join('')
      : '<div class="empty glass">No trips yet — book one from the Book tab.</div>';
    $$('[data-pnr]', box).forEach((b) => (b.onclick = () => { go('pnr'); $('#pnrInput').value = b.dataset.pnr; $('#pnrForm').requestSubmit(); }));
  } catch (err) {
    if (err.status === 401) { setSession(null); return loadTrips(); }
    box.innerHTML = `<div class="empty glass">${esc(err.message)}</div>`;
  }
}

$('#pnrForm').onsubmit = async (e) => {
  e.preventDefault();
  const pnr = $('#pnrInput').value.trim();
  const box = $('#pnrResult');
  try {
    const b = await api('/api/pnr/' + pnr, { title: `PNR ${pnr}` });
    box.innerHTML = `<div class="glass" style="padding:22px;max-width:640px">${ticketHTML(b)}
      <p style="color:var(--muted)">${esc(b.chart ? b.chart.message : '')} · State: <b>${b.state}</b> ${servedBadge(b.servedFrom)}</p>
      ${b.state !== 'ACTIVE' ? ''
        : !state.token ? '<button class="btn danger" id="cancelBtn">Sign in to cancel</button>'
        : state.user === b.user ? '<button class="btn danger" id="cancelBtn">Cancel ticket</button>'
        : '<p class="muted">Only the account that booked this ticket can cancel it.</p>'}</div>`;
    drawCode($('#pnrResult canvas'), b.pnr);
    const cb = $('#cancelBtn');
    if (cb) cb.onclick = async () => {
      if (!state.token) { requireLogin('Sign in as the passenger who booked this ticket to cancel it.').then(() => $('#pnrForm').requestSubmit()).catch(() => {}); return; }
      cb.disabled = true;
      try {
        const r = await withLogin('Sign in to cancel this ticket.', () => api('/api/cancellation/cancel', { method: 'POST', body: { pnr }, title: `Cancel ${pnr}` }));
        toast(`Cancelled. Refund ${inr(r.refund.amount)} (${r.refund.eta}) · charges ${inr(r.charges)}`);
        $('#pnrForm').requestSubmit();
      } catch (err) { toast(err.message, true); cb.disabled = false; }
    };
  } catch (err) { box.innerHTML = `<div class="empty glass">${esc(err.message)}</div>`; }
};

/* ---------------- live status ---------------- */
$('#liveForm').onsubmit = async (e) => {
  e.preventDefault();
  const no = $('#liveTrain').value;
  const box = $('#liveResult');
  try {
    const r = await api('/api/live-status/' + no, { title: `Live status ${no}` });
    const maxKm = r.stops[r.stops.length - 1].km || 1;
    box.innerHTML = `<div class="live glass">
      <div class="live-top"><div><div class="tno">#${r.train}</div><div class="tname">${esc(r.name)}</div></div>
      <span class="delay ${r.delayMins ? '' : 'ontime'}">${r.delayMins ? `Running ${r.delayMins} min late` : 'On time'}</span></div>
      <p style="color:var(--muted)">Last station <b style="color:var(--text)">${r.lastStation}</b>${r.nextStation ? ` · next <b style="color:var(--text)">${r.nextStation}</b>${r.platform ? `, platform ${r.platform.platform}` : ''}` : ' · reached destination'} · updated ${r.updatedAt} IST</p>
      <div class="rail"><div class="fill"></div><div class="loco">🚆</div>
        ${r.stops.map((s) => `<div class="stn ${s.state}" style="left:${(s.km / maxKm) * 100}%"><span>${s.code}<em>${s.arr || s.dep}</em></span></div>`).join('')}
      </div><div style="height:50px"></div></div>`;
    requestAnimationFrame(() => setTimeout(() => {
      $('.rail .fill', box).style.width = r.progress + '%';
      $('.rail .loco', box).style.left = r.progress + '%';
    }, 60));
  } catch (err) { box.innerHTML = `<div class="empty glass">${esc(err.message)}</div>`; }
};

/* ---------------- extras ---------------- */
let extraTab = 'food';
$$('#extraTabs button').forEach((b) => (b.onclick = () => { extraTab = b.dataset.x; $$('#extraTabs button').forEach((x) => x.classList.toggle('on', x === b)); loadExtras(); }));
$('#extraStation').onchange = () => loadExtras();
async function loadExtras() {
  const st = $('#extraStation').value || 'BSB';
  const box = $('#extraResult');
  const card = (i, title, sub, price) => `<div class="card glass" style="animation-delay:${i * 60}ms"><h4>${esc(title)}</h4><div class="muted">${esc(sub)}</div>${price !== undefined ? `<div class="price">${price}</div>` : ''}</div>`;
  try {
    if (extraTab === 'food') {
      const r = await api('/api/food-ordering/menu?station=' + st, { title: 'Food menu' });
      box.innerHTML = r.items.map((x, i) => card(i, x.item, x.veg ? '🟢 Veg' : '🔴 Non-veg', inr(x.price))).join('');
    } else if (extraTab === 'rooms') {
      const r = await api('/api/retiring-rooms/availability?station=' + st, { title: 'Retiring rooms' });
      box.innerHTML = r.rooms.map((x, i) => card(i, x.type, `${r.stationName} · ${x.guests} guest(s) · ${x.available} available`, inr(x.pricePerNight) + '/night')).join('');
    } else if (extraTab === 'tourism') {
      const r = await api('/api/tourism/packages', { title: 'Holiday packages' });
      box.innerHTML = r.map((x, i) => card(i, x.title, `${x.city} · ${x.nights} night(s) · from ${x.station}`, inr(x.price))).join('');
    } else {
      const r = await api('/api/cab-booking/quote?station=' + st + '&km=12', { title: 'Cab quotes' });
      box.innerHTML = r.options.map((x, i) => card(i, x.label, `Pickup ${r.pickup} · ${r.km} km · ETA ${x.etaMins} min`, inr(x.fare))).join('');
    }
  } catch (err) { box.innerHTML = `<div class="empty glass">${esc(err.message)}</div>`; }
}

/* ---------------- admin ---------------- */
async function loadAdmin() {
  try {
    const d = await api('/api/admin/dashboard', { title: 'Ops dashboard' });
    const a = d.analytics || {}, s = d.settlements || {};
    const kpis = [['Bookings', a.bookings || 0], ['Revenue', inr(a.revenue)], ['Cancellations', a.cancellations || 0], ['Refunds', inr(a.refunds)],
      ['Ledger entries', s.entries || 0], ['Services up', d.platform ? `${d.platform.up}/${d.platform.total}` : '—'], ['Avg rating', d.feedback ? d.feedback.average || '—' : '—']];
    $('#adminKpis').innerHTML = kpis.map(([k, v], i) => `<div class="kpi glass card" style="animation-delay:${i * 60}ms"><span>${k}</span><b>${v}</b></div>`).join('');
    const recent = (d.audit && d.audit.recent) || [];
    $('#adminAudit').innerHTML = recent.length ? recent.map((e) => `<div class="audit"><div><b>${esc(e.action)}</b> ${esc(e.resource)} · ${esc(e.actor)}</div><span>${esc(e.at)}</span></div>`).join('') : '<p style="color:var(--muted)">No activity yet.</p>';
    const by = a.byClass || {}, max = Math.max(1, ...Object.values(by));
    $('#adminClasses').innerHTML = Object.keys(by).length ? Object.entries(by).map(([c, n]) => `<div class="bar"><b>${c}</b><div style="width:${(n / max) * 100}%"></div><span>${n}</span></div>`).join('') : '<p style="color:var(--muted)">Book a few tickets to see this fill up.</p>';
  } catch (err) { $('#adminKpis').innerHTML = `<div class="empty glass">${esc(err.message)}</div>`; }
}

/* ---------------- confetti ---------------- */
function confetti() {
  const c = $('#confetti'), ctx = c.getContext('2d');
  c.width = innerWidth; c.height = innerHeight;
  const colors = ['#00ffa3', '#00e5ff', '#ff2bd6', '#ffc53d', '#ffffff'];
  const parts = Array.from({ length: 180 }, () => ({ x: innerWidth / 2, y: innerHeight / 2, vx: (Math.random() - 0.5) * 16, vy: Math.random() * -16 - 4,
    s: Math.random() * 6 + 3, c: colors[(Math.random() * colors.length) | 0], r: Math.random() * 6, vr: (Math.random() - 0.5) * 0.4 }));
  let f = 0;
  (function frame() {
    ctx.clearRect(0, 0, c.width, c.height);
    parts.forEach((p) => { p.vy += 0.45; p.x += p.vx; p.y += p.vy; p.r += p.vr; ctx.save(); ctx.translate(p.x, p.y); ctx.rotate(p.r); ctx.fillStyle = p.c; ctx.fillRect(-p.s / 2, -p.s / 2, p.s, p.s * 0.6); ctx.restore(); });
    if (f++ < 150) requestAnimationFrame(frame); else ctx.clearRect(0, 0, c.width, c.height);
  })();
}

/* ---------------- boot ---------------- */
function countUp() {
  $$('[data-count]').forEach((el) => {
    const target = Number(el.dataset.count), start = performance.now();
    (function step(now) { const p = Math.min(1, (now - start) / 1400); el.textContent = Math.round(target * (1 - Math.pow(1 - p, 3))); if (p < 1) requestAnimationFrame(step); })(start);
  });
}

(async function boot() {
  countUp();
  const d = new Date(Date.now() + 86400000);
  $('#date').value = d.toISOString().slice(0, 10);
  fromAC = stationInput('#from', '#fromSuggest');
  toAC = stationInput('#to', '#toSuggest');
  try {
    state.services = await (await fetch('/services.json')).json();
    state.services.forEach((s) => (state.byName[s.name] = s));
    buildMesh();
  } catch { /* mesh stays empty */ }
  fetch('/runtime.json').then((r) => r.json()).then((rt) => {
    $('#runtimeMeta').textContent = `cluster: ${rt.cluster} · env: ${rt.environment} · v${rt.version} · pod ${rt.pod}`;
  }).catch(() => {});
  api('/api/config/flags', { quiet: true }).then((f) => {
    if (f.saleBanner) { $('#saleText').textContent = `${f.saleBanner}   ✦   ${f.saleBanner}   ✦   ${f.saleBanner}`; $('#sale').classList.remove('hidden'); }
  }).catch(() => {});
  try {
    state.stations = await api('/api/stations', { quiet: true });
    const byCode = Object.fromEntries(state.stations.map((s) => [s.code, s]));
    fromAC.set(byCode.NDLS); toAC.set(byCode.BSB);
    const opts = state.stations.map((s) => `<option value="${s.code}">${esc(s.name)} (${s.code})</option>`).join('');
    $('#extraStation').innerHTML = opts;
    $('#extraStation').value = 'BSB';
    const trains = await api('/api/trains', { quiet: true });
    $('#liveTrain').innerHTML = trains.map((t) => `<option value="${t.number}">${t.number} · ${esc(t.name)}</option>`).join('');
  } catch (err) { toast('Could not reach the platform: ' + err.message, true); }
  restoreSession();
  pollHealth();
  setInterval(pollHealth, 5000);
  go(location.hash.slice(1) || 'search');
})();
