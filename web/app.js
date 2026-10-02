/* Energofish Partnerhírlevél-generátor – felület */
'use strict';

const TOKEN = document.querySelector('meta[name="api-token"]').content;
const VERSION = document.querySelector('meta[name="app-version"]').content;
const PREHEADER_FILLER = new RegExp('[' + String.fromCharCode(0x2007, 0x034F, 0x200C, 0xA0) + ']', 'g');
const NBSP = String.fromCharCode(0xA0);

const S = {
  fields: [], groups: [], templates: [], tokens: [], defaults: {}, sample: null,
  state: null, excel: null,
  issues: { content: [], partners: [], counts: {}, blocked: 0 },
  pStatus: new Map(),
  sel: new Set(), pv: -1, device: 'desktop', tab: 'adatok', search: '',
  lastResult: null, imgResults: null, imgBusy: false, genBusy: false,
  mode: 'webview', config: '', pvSeq: 0, syncTimer: null, syncing: null,
  feed: null, feedURL: '',
};

/* ------------------------------------------------------------------ segédek */

// Az append/replaceChildren fogadjon tömböt, és hagyja ki a null/false elemeket.
for (const m of ['append', 'prepend', 'replaceChildren']) {
  const orig = Element.prototype[m];
  Element.prototype[m] = function (...a) {
    return orig.apply(this, a.flat(Infinity).filter(x => x != null && x !== false));
  };
}

const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => Array.from(r.querySelectorAll(s));
const PROPS = new Set(['value', 'checked', 'disabled', 'selected', 'indeterminate', 'rows', 'type', 'placeholder', 'spellcheck']);

function h(tag, props, ...kids) {
  const e = document.createElement(tag);
  if (props) {
    for (const [k, v] of Object.entries(props)) {
      if (v == null || v === false) continue;
      if (k === 'class') e.className = v;
      else if (k === 'text') e.textContent = v;
      else if (k === 'html') e.innerHTML = v;
      else if (k.startsWith('on') && typeof v === 'function') e.addEventListener(k.slice(2), v);
      else if (k === 'style' && typeof v === 'object') Object.assign(e.style, v);
      else if (k === 'dataset') Object.assign(e.dataset, v);
      else if (PROPS.has(k)) e[k] = v;
      else e.setAttribute(k, v === true ? '' : v);
    }
  }
  for (const kid of kids.flat(Infinity)) {
    if (kid == null || kid === false) continue;
    e.append(kid.nodeType ? kid : document.createTextNode(String(kid)));
  }
  return e;
}

const ICONS = {
  sheet: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/><path d="M8 12h8v6H8zM12 12v6M8 15h8"/>',
  upload: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="M17 8l-5-5-5 5"/><path d="M12 3v12"/>',
  download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="M7 10l5 5 5-5"/><path d="M12 15V3"/>',
  refresh: '<path d="M21 12a9 9 0 1 1-2.64-6.36"/><path d="M21 3v6h-6"/>',
  external: '<path d="M15 3h6v6"/><path d="M10 14L21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>',
  x: '<path d="M18 6L6 18M6 6l12 12"/>',
  check: '<path d="M20 6L9 17l-5-5"/>',
  alert: '<path d="M10.3 3.9L1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"/><path d="M12 9v4M12 17h.01"/>',
  error: '<circle cx="12" cy="12" r="10"/><path d="M15 9l-6 6M9 9l6 6"/>',
  info: '<circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/>',
  monitor: '<rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>',
  phone: '<rect x="6" y="2" width="12" height="20" rx="2"/><path d="M11 18h2"/>',
  folder: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>',
  mail: '<rect x="2" y="4" width="20" height="16" rx="2"/><path d="M22 6l-10 7L2 6"/>',
  image: '<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/>',
  pen: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>',
  grid: '<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>',
  poll: '<circle cx="12" cy="12" r="10"/><path d="M9.1 9a3 3 0 0 1 5.8 1c0 2-3 3-3 3"/><path d="M12 17h.01"/>',
  user: '<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>',
  users: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>',
  footer: '<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 15h18"/>',
  link: '<path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  trash: '<path d="M3 6h18"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>',
  up: '<path d="M12 19V5M5 12l7-7 7 7"/>',
  down: '<path d="M12 5v14M19 12l-7 7-7-7"/>',
  chev: '<path d="M6 9l6 6 6-6"/>',
  left: '<path d="M15 18l-6-6 6-6"/>',
  right: '<path d="M9 18l6-6-6-6"/>',
  search: '<circle cx="11" cy="11" r="8"/><path d="M21 21l-4.3-4.3"/>',
  dots: '<circle cx="12" cy="5" r="1.2"/><circle cx="12" cy="12" r="1.2"/><circle cx="12" cy="19" r="1.2"/>',
  save: '<path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><path d="M17 21v-8H7v8M7 3v5h8"/>',
  open: '<path d="M4 22h14a2 2 0 0 0 2-2V7.5L14.5 2H6a2 2 0 0 0-2 2v4"/><path d="M14 2v6h6"/><path d="M2 15h10M9 18l3-3-3-3"/>',
  reset: '<path d="M3 12a9 9 0 1 0 2.64-6.36"/><path d="M3 3v6h6"/>',
  zap: '<path d="M13 2L3 14h9l-1 8 10-12h-9z"/>',
  power: '<path d="M18.4 6.6a9 9 0 1 1-12.8 0"/><path d="M12 2v10"/>',
  box: '<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><path d="M3.3 7L12 12l8.7-5M12 22V12"/>',
  shield: '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="M9 12l2 2 4-4"/>',
  table: '<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M9 21V9"/>',
  eye: '<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>',
  tablet: '<rect x="4" y="2" width="16" height="20" rx="2"/><path d="M11 18h2"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>',
  copy: '<rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>',
};

function icon(name, size = 18) {
  const s = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  s.setAttribute('viewBox', '0 0 24 24');
  s.setAttribute('width', size);
  s.setAttribute('height', size);
  s.setAttribute('class', 'ic');
  s.setAttribute('aria-hidden', 'true');
  s.innerHTML = ICONS[name] || '';
  return s;
}

async function api(path, body, opts = {}) {
  const headers = Object.assign({ 'X-Token': TOKEN }, opts.raw ? {} : { 'Content-Type': 'application/json' }, opts.headers || {});
  let res;
  try {
    res = await fetch(path, { method: 'POST', headers, body: opts.raw ? body : JSON.stringify(body || {}) });
  } catch (e) {
    throw new Error('A program nem válaszol. Ha bezártad az ablakát, indítsd újra.');
  }
  let data = null;
  try { data = await res.json(); } catch (e) { /* üres */ }
  if (!res.ok || (data && data.error)) throw new Error((data && data.error) || ('Váratlan hiba (' + res.status + ')'));
  return data || {};
}

function norm(s) {
  return String(s || '').toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '');
}

function initials(name) {
  const words = String(name || '').split(/\s+/).filter(w => w && !w.endsWith('.'));
  if (!words.length) return '';
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (words[0][0] + words[1][0]).toUpperCase();
}

function assetsBase() {
  const b = String((S.state && S.state.content['assets.base']) || '').trim().replace(/\/+$/, '');
  return b || '/assets';
}

function resolveUrl(u, code) {
  let v = String(u || '').trim();
  if (!v) return '';
  v = v.replace(/\{assets\}/gi, assetsBase());
  if (code != null) v = v.replace(/\{cikksz[aá]m\}/gi, encodeURIComponent(code));
  return v;
}

// A változók behelyettesítése az előnézetben látható partner adataival (karakterszámláláshoz).
function expandPreview(text) {
  const p = (S.excel && S.excel.partners[S.pv]) || S.sample || {};
  const map = {
    nev: p.name, email: p.email, ceg: p.company, kepviselo: p.repName, kepviseloemail: p.repEmail,
    kepviselotelefon: p.repPhone, terulet: p.repRegion, termekszam: String(S.state.products.filter(x => x.on).length),
  };
  for (const [k, v] of Object.entries(p.extra || {})) if (!(k in map)) map[k] = v;
  for (const t of (S.excel && S.excel.tokens) || []) { const k = t.token.slice(1, -1); if (!(k in map)) map[k] = ''; }
  return String(text || '').replace(/\{([^{}\n]{1,40})\}/g, (m, k) => {
    const n = norm(k).replace(/[^a-z0-9]/g, '');
    return n in map ? (map[n] || '') : m;
  });
}

function validPhoto(u) {
  const v = String(u || '').trim();
  return /^https?:\/\//i.test(v) || /^\{assets\}/i.test(v) || v.startsWith('/assets');
}

function fmtTime(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '';
  const pad = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}. ${pad(d.getMonth() + 1)}. ${pad(d.getDate())}. ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function formatPrice(s) {
  const m = String(s || '').trim().match(/^(\d{1,12})([.,]\d+)?\s*(Ft|HUF|ft)?$/);
  if (!m) return String(s || '').trim();
  const n = Math.round(parseFloat(m[1] + (m[2] || '').replace(',', '.')));
  return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, NBSP) + NBSP + 'Ft';
}

function plural(n, word) { return `${n} ${word}`; }

function debounce(fn, ms) {
  let t;
  return (...a) => { clearTimeout(t); t = setTimeout(() => fn(...a), ms); };
}

function ls(key, val) {
  try {
    if (val === undefined) return JSON.parse(localStorage.getItem('ef.' + key));
    localStorage.setItem('ef.' + key, JSON.stringify(val));
  } catch (e) { return null; }
  return null;
}

/* ------------------------------------------------------------------ toast, modális */

function toast(msg, kind = 'info', opts = {}) {
  const box = $('#toasts');
  const ic = { ok: 'check', err: 'error', warn: 'alert', info: 'info' }[kind] || 'info';
  const body = h('div', null, h('div', { text: msg }));
  if (opts.url) body.append(h('div', { class: 'u', text: opts.url }));
  if (opts.actions && opts.actions.length) {
    body.append(h('div', { class: 'act' }, opts.actions.map(a => h('button', { text: a.label, onclick: () => { a.fn(); close(); } }))));
  }
  const t = h('div', { class: 'toast ' + kind }, icon(ic, 18), body, h('button', { class: 'x', title: 'Bezárás', onclick: () => close() }, icon('x', 16)));
  function close() { t.remove(); }
  box.append(t);
  while (box.children.length > 4) box.firstChild.remove();
  const timeout = opts.timeout != null ? opts.timeout : (kind === 'err' ? 9000 : 5000);
  if (timeout) setTimeout(close, timeout);
  return t;
}

function confirmBox(title, text, okLabel = 'Rendben', danger = false) {
  return new Promise(resolve => {
    const done = v => { bg.remove(); document.removeEventListener('keydown', key); resolve(v); };
    const key = e => { if (e.key === 'Escape') done(false); };
    const bg = h('div', { class: 'modal-bg', onclick: e => { if (e.target === bg) done(false); } },
      h('div', { class: 'modal', role: 'dialog' },
        h('div', { class: 'mh', text: title }),
        h('div', { class: 'mb', text: text }),
        h('div', { class: 'mf' },
          h('button', { class: 'btn btn-ghost', text: 'Mégse', onclick: () => done(false) }),
          h('button', { class: 'btn ' + (danger ? 'btn-dark' : 'btn-primary'), text: okLabel, onclick: () => done(true) }))));
    document.addEventListener('keydown', key);
    document.body.append(bg);
    $('.mf .btn:last-child', bg).focus();
  });
}

async function busy(btn, fn) {
  if (btn.disabled) return;
  const old = Array.from(btn.childNodes);
  btn.disabled = true;
  btn.replaceChildren(h('span', { class: 'spin' }), h('span', { text: btn.dataset.busy || 'Egy pillanat…' }));
  try { return await fn(); } finally {
    btn.disabled = false;
    btn.replaceChildren(...old);
  }
}

/* ------------------------------------------------------------------ állapot szinkron */

function syncSoon(ms = 350) {
  clearTimeout(S.syncTimer);
  S.syncTimer = setTimeout(syncNow, ms);
}

async function syncNow() {
  clearTimeout(S.syncTimer);
  S.syncing = api('/api/state', S.state)
    .then(r => {
      if (r && r.issues) setIssues(r.issues);
      refreshPreview();
    })
    .catch(e => toast('Mentési hiba: ' + e.message, 'err'));
  return S.syncing;
}

function setIssues(iss) {
  S.issues = iss;
  S.pStatus = new Map();
  const rank = { error: 3, warn: 2, info: 1 };
  for (const i of iss.partners) {
    if (i.scope !== 'partner') continue;
    const cur = S.pStatus.get(i.index);
    if (!cur || rank[i.level] > rank[cur.level]) S.pStatus.set(i.index, { level: i.level, msgs: (cur ? cur.msgs : []).concat(i.message) });
    else cur.msgs.push(i.message);
  }
  updateFieldIssues();
  updateProductIssues();
  renderSteps();
  if (S.tab === 'ellenorzes') renderCheck();
  if (S.tab === 'generalas') renderGenSummary();
}

function isBlocked(i) {
  const st = S.pStatus.get(i);
  return !!(st && st.level === 'error');
}

function contentIssues(scope) { return S.issues.content.filter(i => i.scope === scope); }

function curTpl() {
  return S.templates.find(t => t.id === S.state.template) || S.templates[0] || {};
}

/* ------------------------------------------------------------------ keret */

const STEPS = [
  { id: 'adatok', label: 'Adatok' },
  { id: 'tartalom', label: 'Tartalom' },
  { id: 'termekek', label: 'Termékek' },
  { id: 'ellenorzes', label: 'Ellenőrzés' },
  { id: 'generalas', label: 'Generálás' },
];

function buildShell() {
  const nav = $('#steps');
  STEPS.forEach((s, i) => {
    nav.append(h('button', { class: 'step', dataset: { step: s.id }, onclick: () => setTab(s.id), title: `${s.label} (Ctrl+${i + 1})` },
      h('span', { class: 'num', text: i + 1 }), h('span', { class: 'lbl', text: s.label }), h('span', { class: 'badge' })));
  });
  $('#menuBtn').append(icon('dots', 20));
  $('#settingsBtn').append(icon('gear', 19));
  $('#settingsBtn').addEventListener('click', () => openSettings());
  $('#menuBtn').addEventListener('click', e => { e.stopPropagation(); toggleMenu(); });
  document.addEventListener('click', e => { if (!e.target.closest('#menu')) $('#menu').classList.add('hidden'); });
  document.addEventListener('keydown', e => {
    if ((e.ctrlKey || e.metaKey) && e.key >= '1' && e.key <= '5') { e.preventDefault(); setTab(STEPS[+e.key - 1].id); }
  });
  $('#pvToggle').append(icon('eye'), ' Előnézet');
  $('#pvToggle').addEventListener('click', () => $('#preview').classList.toggle('show'));
  buildPreview();
  setupDrop();
  setupContextMenu();
}

function setTab(id) {
  S.tab = id;
  ls('tab', id);
  $$('.pane').forEach(p => p.classList.toggle('active', p.dataset.pane === id));
  $$('.step').forEach(b => b.classList.toggle('active', b.dataset.step === id));
  if (id === 'ellenorzes') renderCheck();
  if (id === 'generalas') renderGenerate();
  if (id === 'tartalom') autosizeAll(pane('tartalom'));
  $('#editor').scrollTop = 0;
}

function renderSteps() {
  const set = (id, text, cls) => {
    const b = $(`.step[data-step="${id}"] .badge`);
    if (!b) return;
    b.textContent = text == null ? '' : text;
    b.className = 'badge' + (text != null && text !== '' ? ' show' : '') + (cls ? ' ' + cls : '');
  };
  const ex = S.issues.partners.filter(i => i.scope === 'excel' && i.level === 'error').length;
  set('adatok', S.excel ? (ex ? '!' : S.excel.partners.length) : null, ex ? 'err' : '');
  const ce = contentIssues('content');
  const cErr = ce.filter(i => i.level === 'error').length;
  const cWarn = ce.filter(i => i.level === 'warn').length;
  set('tartalom', cErr || cWarn || null, cErr ? 'err' : (cWarn ? 'warn' : ''));
  const pe = contentIssues('product').filter(i => i.level === 'error').length;
  const selP = S.state.products.filter(p => p.on).length;
  set('termekek', pe ? pe : selP, pe ? 'err' : '');
  const c = S.issues.counts || {};
  set('ellenorzes', c.error ? c.error : (c.warn ? c.warn : '✓'), c.error ? 'err' : (c.warn ? 'warn' : 'ok'));
}

function toggleMenu() {
  const m = $('#menu');
  if (!m.classList.contains('hidden')) { m.classList.add('hidden'); return; }
  const item = (ic, label, fn) => h('button', { onclick: () => { m.classList.add('hidden'); fn(); } }, icon(ic), label);
  m.replaceChildren(
    item('gear', 'Beállítások…', () => openSettings()),
    h('hr'),
    item('save', 'Tartalom mentése fájlba…', exportContent),
    item('open', 'Tartalom betöltése fájlból…', importContent),
    item('reset', 'Közös tartalom visszaállítása a mintára…', resetContent),
    item('trash', 'Termékek és partnerek törlése…', () => resetData(true, true)),
    item('reset', 'Visszaállítás alapállapotba…', () => openFactoryReset()),
    h('hr'),
    item('users', 'Partnerek a B2B partnertörzsből…', () => { setTab('adatok'); openPartnerSet(); }),
    item('link', 'Partnertörzs-források (linkek)…', () => openSources()),
    h('hr'),
    item('download', 'Minta Excel mentése…', saveDemo),
    item('plus', 'Sablon hozzáadása…', importTemplates),
    item('folder', 'Hozzáadott sablonok mappája', () => api('/api/templates/folder').catch(e => toast(e.message, 'err'))),
    item('folder', 'Beállítások és napló mappája', () => openPath(S.config ? S.config.replace(/[\\/][^\\/]*$/, '') : '')),
    S.mode === 'browser' ? [h('hr'), item('power', 'Kilépés a programból', quitApp)] : null,
    h('div', { class: 'ver', text: `Energofish Partnerhírlevél-generátor · ${VERSION}` }));
  m.classList.remove('hidden');
}

function secHead(kicker, title, text) {
  return h('div', { class: 'sec-head' }, h('div', { class: 'kicker', text: kicker }), h('h2', { text: title }), text ? h('p', { text }) : null);
}

function pane(id) { return $(`.pane[data-pane="${id}"]`); }

async function openPath(path, reveal) {
  if (!path) return;
  try { await api('/api/open', { path, reveal: !!reveal }); } catch (e) { toast(e.message, 'err'); }
}

/* ------------------------------------------------------------------ 1. Adatok */

function renderData() {
  const p = pane('adatok');
  p.replaceChildren(secHead('01 / Adatforrás', 'Partnerek',
    'A partnerek jöhetnek a B2B partnertörzsből (célcsoport, képviselő, megye, besorolás és más tulajdonságok szerint összeállított halmaz) vagy Excel-fájlból. Ami minden partnernél ugyanaz, azt a Tartalom lépésben adod meg.'));
  if (!S.excel) {
    p.append(b2bCard(), h('div', { class: 'or-sep' }, h('span', { text: 'vagy Excel-fájlból' })), dropZone(), formatCard());
    return;
  }
  p.append(S.excel.source === 'b2b' ? b2bFileCard() : fileCard(), statsRow());
  for (const i of S.issues.partners.filter(i => i.scope === 'excel')) p.append(noteFor(i));
  p.append(partnerCard());
}

function dropZone() {
  const im = S.import || {};
  return h('div', { class: 'drop', id: 'dropZone' },
    h('div', { class: 'big-ic' }, icon('sheet', 30)),
    h('h3', null, 'Import Excel: ', h('span', { class: 'drop-name', text: im.excelName || '' })),
    h('p', { class: 'drop-path', text: im.exists ? `Megvan: ${im.path} (módosítva: ${fmtTime(im.modTime)})` : `Itt keresem: ${im.path || ''} – még nincs ott.` }),
    h('div', { class: 'btn-row', style: { justifyContent: 'center' } },
      im.exists ? h('button', { class: 'btn btn-primary', onclick: e => busy(e.currentTarget, importExcel) }, icon('upload'), 'Betöltés') : null,
      h('button', { class: im.exists ? 'btn btn-outline' : 'btn btn-primary', title: 'Csak ezzel a névvel olvasható be; a mappája bárhol lehet', onclick: e => busy(e.currentTarget, browseExcel) }, icon('open'), 'Tallózás…'),
      h('button', { class: 'btn btn-outline', onclick: e => busy(e.currentTarget, saveDemo) }, icon('download'), 'Minta Excel mentése'),
      h('button', { class: 'btn btn-ghost', onclick: () => openSettings('import') }, icon('gear', 16), 'Név és mappa…')),
    h('p', { class: 'p-sub', style: { margin: '10px 0 0' }, text: 'A program csak ezzel a névvel olvas be Excelt (behúzva, tallózva vagy a fenti helyről). A név és a mappa a Beállításokban módosítható.' }));
}

async function importExcel() {
  try { applyExcel(await api('/api/excel/import')); } catch (e) { toast(e.message, 'err', { timeout: 12000 }); }
}

function formatCard() {
  const col = (name, req, note) => h('li', null, h('span', { class: 'swatch' + (req ? ' req' : '') }), h('span', { text: name }), h('span', { class: 'opt', text: req ? 'kötelező' : (note || 'opcionális') }));
  return h('div', { class: 'card card-pad', style: { marginTop: '16px' } },
    h('div', { class: 'card-title' }, icon('table'), 'Milyen legyen az Excel?'),
    h('p', { class: 'card-sub', text: 'A program a fejléc szövegéből ismeri fel az oszlopokat, a sorrend tetszőleges. A „Minta Excel mentése” gomb kész, kitöltött példát ad.' }),
    h('div', { class: 'cols-grid' },
      h('div', null, h('div', { class: 'label-caps', text: '1. munkalap · Partnerek' }), h('ul', { class: 'col-list' },
        col('Partner e-mail', true), col('Partner neve', true, ''), col('Területi képviselő', true), col('Képviselő kép link'),
        col('Képviselő telefon'), col('Képviselő e-mail'), col('Képviselő területe'), col('Cégnév'))),
      h('div', null, h('div', { class: 'label-caps', text: '2. munkalap · Termékek' }), h('ul', { class: 'col-list' },
        col('Cikkszám', true), col('Cikknév', true), col('Cikk kép link'), col('Gomb link'),
        col('Rövid leírás'), col('Ár'), col('Akció')))),
    h('div', { class: 'note cream', style: { margin: '16px 0 0' } }, icon('info'),
      h('div', null, 'Bármely további partner-oszlop változóként is használható (pl. „Partnerkód” → ', h('span', { class: 'tok', text: '{partnerkod}' }),
        '). Ha egy oszlop fejléce sablonkulcs (pl. ', h('span', { class: 'tok', text: 'note.body' }), '), az adott partnernél felülírja a közös szöveget.')));
}

function fileCard() {
  const ex = S.excel;
  const abs = /^([a-zA-Z]:[\\/]|\/|\\\\)/.test(ex.path || '');
  const sheets = [ex.partnerSheet, ex.productSheet].filter(Boolean).join(' · ');
  return h('div', { class: 'card card-pad' },
    h('div', { class: 'file-card' },
      h('div', { class: 'file-ic' }, icon('sheet', 26)),
      h('div', { style: { minWidth: 0, flex: 1 } },
        h('div', { class: 'file-name', text: ex.fileName }),
        abs ? h('div', { class: 'file-path', text: ex.path }) : null,
        h('div', { class: 'file-meta', text: `Betöltve: ${fmtTime(ex.loadedAt)}${sheets ? ' · Munkalapok: ' + sheets : ''}` })),
      h('button', { class: 'btn btn-ghost btn-sm', title: 'Lista bezárása', onclick: closeExcel, style: { alignSelf: 'flex-start' } }, icon('x', 16))),
    h('div', { class: 'file-actions' },
        abs ? h('button', { class: 'btn btn-primary btn-sm', title: 'Mentsd az Excelt, majd töltsd újra', onclick: e => busy(e.currentTarget, reloadExcel) }, icon('refresh', 16), 'Újratöltés') : null,
        abs ? h('button', { class: 'btn btn-outline btn-sm', onclick: () => api('/api/excel/open').catch(e => toast(e.message, 'err')) }, icon('external', 16), 'Megnyitás Excelben') : null,
        h('button', { class: 'btn btn-ghost btn-sm', onclick: e => busy(e.currentTarget, browseExcel) }, icon('open', 16), 'Másik fájl…'),
        h('button', { class: 'btn btn-ghost btn-sm', title: 'Partnerek a B2B partnertörzsből (a webshop feliratkozói), Excel helyett', onclick: () => openPartnerSet() }, icon('users', 16), 'B2B partnertörzsből…'),
        abs ? h('span', { class: 'p-sub', style: { marginLeft: 'auto' }, text: 'Az Excel módosítása után mentsd a fájlt, majd Újratöltés.' }) : null));
}

function statsRow() {
  const ex = S.excel;
  const blocked = S.issues.blocked || 0;
  const reps = ex.reps || 0;
  return h('div', { class: 'stats' },
    h('div', { class: 'stat accent' }, h('b', { text: ex.partners.length }), h('span', { text: 'partner' })),
    h('div', { class: 'stat' }, h('b', { text: reps }), h('span', { text: 'képviselő' })),
    h('div', { class: 'stat' }, h('b', { text: ex.source === 'b2b' ? S.state.products.filter(x => x.on).length : (ex.products || []).length }), h('span', { text: 'termék' })),
    h('div', { class: 'stat' + (blocked ? ' err' : '') }, h('b', { text: blocked }), h('span', { text: 'hibás, kimarad' })));
}

function noteFor(i) {
  const map = { error: ['err', 'error'], warn: ['warn', 'alert'], info: ['info', 'info'] };
  const [cls, ic] = map[i.level] || map.info;
  return h('div', { class: 'note ' + cls }, icon(ic), h('div', { text: i.message }));
}

function partnerCard() {
  const card = h('div', { class: 'card' });
  const search = h('input', { type: 'search', placeholder: 'Keresés névre, e-mailre, cégre, képviselőre…', value: S.search });
  search.addEventListener('input', debounce(() => { S.search = search.value; renderPartnerRows(); }, 150));
  card.append(
    h('div', { class: 'table-tools' },
      h('label', { class: 'search' }, icon('search', 16), search),
      h('span', { class: 'sel-info', id: 'selInfo' })),
    h('div', { class: 'table-wrap', id: 'ptWrap' },
      h('table', { class: 'pt' },
        h('thead', null, h('tr', null,
          h('th', { class: 'c-chk' }, h('input', { type: 'checkbox', id: 'selAll', title: 'Összes (látható) kijelölése', onchange: e => selectVisible(e.target.checked) })),
          h('th', { class: 'c-row', text: 'Sor' }), h('th', { text: 'Partner' }), h('th', { text: 'Területi képviselő' }), h('th', { class: 'c-st' }))),
        h('tbody', { id: 'ptBody' }))),
    columnsDetails());
  setTimeout(renderPartnerRows);
  return card;
}

function visiblePartners() {
  const q = norm(S.search).trim();
  const out = [];
  S.excel.partners.forEach((p, i) => {
    if (!q || norm([p.name, p.email, p.company, p.repName, p.repRegion, ...Object.values(p.extra || {})].join(' ')).includes(q)) out.push(i);
  });
  return out;
}

function avatar(name, photo, size) {
  const a = h('div', { class: 'avatar', text: initials(name) });
  if (size) Object.assign(a.style, { width: size + 'px', height: size + 'px' });
  if (photo && validPhoto(photo)) {
    const url = resolveUrl(photo);
    const img = new Image();
    img.onload = () => { a.textContent = ''; a.style.backgroundImage = `url("${url.replace(/"/g, '%22')}")`; };
    img.src = url;
  }
  return a;
}

function renderPartnerRows() {
  const body = $('#ptBody');
  if (!body) return;
  const all = visiblePartners();
  const vis = all.slice(0, MAX_PARTNER_ROWS);
  const rows = vis.map(i => {
    const p = S.excel.partners[i];
    const st = S.pStatus.get(i);
    const stEl = st
      ? h('span', { class: 'st ' + (st.level === 'error' ? 'err' : st.level), title: st.msgs.join('\n') }, icon(st.level === 'error' ? 'error' : (st.level === 'warn' ? 'alert' : 'info'), 15))
      : h('span', { class: 'st ok', title: 'Rendben' }, icon('check', 15));
    const chk = h('input', { type: 'checkbox', checked: S.sel.has(i), onclick: e => e.stopPropagation(), onchange: e => { e.target.checked ? S.sel.add(i) : S.sel.delete(i); updateSelInfo(); tr.classList.toggle('off', !e.target.checked); } });
    const tr = h('tr', { class: 'row' + (i === S.pv ? ' current' : '') + (S.sel.has(i) ? '' : ' off'), dataset: { i }, onclick: () => setPreviewPartner(i) },
      h('td', { class: 'c-chk' }, chk),
      h('td', { class: 'c-row', text: p.row }),
      h('td', null, h('div', { class: 'p-name', text: p.name || p.company || '(név nélkül)' }), h('div', { class: 'p-sub', text: (p.email || '– nincs e-mail –') + (p.extra && p.extra.nazon ? ' · ' + p.extra.nazon : '') }), p.company && p.name && p.company !== p.name ? h('div', { class: 'p-sub', text: p.company }) : null),
      h('td', null, p.repName ? h('div', { class: 'rep' }, avatar(p.repName, p.repPhoto), h('div', { style: { minWidth: 0 } }, h('div', { class: 'rep-name', text: p.repName }), h('div', { class: 'rep-sub', text: p.repRegion || p.repEmail || '' }))) : h('span', { class: 'p-sub', text: '–' })),
      h('td', { class: 'c-st' }, stEl));
    return tr;
  });
  body.replaceChildren(...rows);
  if (!rows.length) body.append(h('tr', null, h('td', { colspan: 5, class: 'empty', text: 'Nincs találat.' })));
  if (all.length > vis.length) body.append(h('tr', null, h('td', { colspan: 5, class: 'empty', text: `… és még ${all.length - vis.length} partner. Keress névre, e-mailre vagy képviselőre a szűréshez.` })));
  updateSelInfo();
}

function updateSelInfo() {
  const el = $('#selInfo');
  if (!el || !S.excel) return;
  el.textContent = `${S.sel.size} / ${S.excel.partners.length} kijelölve`;
  const vis = visiblePartners();
  const all = $('#selAll');
  if (all) {
    const n = vis.filter(i => S.sel.has(i)).length;
    all.checked = n > 0 && n === vis.length;
    all.indeterminate = n > 0 && n < vis.length;
  }
  if (S.tab === 'generalas') renderGenSummary();
}

function selectVisible(on) {
  for (const i of visiblePartners()) on ? S.sel.add(i) : S.sel.delete(i);
  renderPartnerRows();
}

function columnsDetails() {
  const ex = S.excel;
  const item = c => h('div', { class: 'map-item' + (c.field ? '' : ' unused') },
    h('span', { class: 'col', text: c.letter }), h('span', { class: 'hdr', text: c.header, title: c.header }), h('span', { class: 'arrow', text: '→' }),
    h('span', { class: 'to', text: c.field ? c.label : 'nem használt', title: c.label }));
  return h('details', { class: 'cols' },
    h('summary', null, icon('table', 16), `Felismert oszlopok (${ex.partnerSheet || '–'}${ex.productSheet ? ', ' + ex.productSheet : ''})`, h('span', { style: { marginLeft: 'auto' } }, icon('chev', 16))),
    h('div', { class: 'label-caps', style: { padding: '0 22px 6px' }, text: 'Partnerek' }),
    h('div', { class: 'map-list' }, (ex.partnerColumns || []).map(item)),
    ex.productColumns && ex.productColumns.length ? [h('div', { class: 'label-caps', style: { padding: '4px 22px 6px' }, text: 'Termékek' }), h('div', { class: 'map-list' }, ex.productColumns.map(item))] : null);
}

function applyExcel(r, quiet) {
  if (r.import) S.import = r.import;
  S.excel = normalizeExcel(r.excel || null);
  if (r.products) S.state.products = r.products;
  setIssues(r.issues);
  S.sel = new Set(S.excel ? S.excel.partners.map((_, i) => i) : []);
  S.pv = S.excel && S.excel.partners.length ? 0 : -1;
  S.search = '';
  renderData();
  renderProducts();
  refreshTokenNote();
  renderPreviewSelect();
  refreshPreview();
  if (S.tab === 'generalas') renderGenerate();
  if (!quiet && S.excel) {
    const errs = S.issues.partners.filter(i => i.level === 'error').length;
    toast(`Betöltve: ${S.excel.partners.length} partner, ${(S.excel.products || []).length} termék` + (errs ? ` · ${errs} hiba` : ''), errs ? 'warn' : 'ok');
  }
}

async function browseExcel() {
  try {
    const r = await api('/api/excel/browse');
    if (r.cancelled) return;
    applyExcel(r);
  } catch (e) { toast(e.message, 'err'); }
}

async function reloadExcel() {
  try { applyExcel(await api('/api/excel/reload')); } catch (e) { toast(e.message, 'err'); }
}

async function closeExcel() {
  const b2b = S.excel && S.excel.source === 'b2b';
  if (!await confirmBox('Lista bezárása', b2b ? 'A partnerhalmaz kikerül a hírlevélből (a partnertörzs és a mentett halmazok megmaradnak). A termékek is megmaradnak.' : 'A partnerlista kikerül a programból (az Excel-fájl nem változik). A termékek megmaradnak.', 'Bezárás')) return;
  try {
    const r = await api('/api/excel/close');
    S.excel = null;
    setIssues(r.issues);
    S.sel = new Set(); S.pv = -1;
    renderData(); refreshTokenNote(); renderPreviewSelect(); refreshPreview();
  } catch (e) { toast(e.message, 'err'); }
}

async function saveDemo() {
  try {
    const r = await api('/api/demo/save');
    if (r.cancelled) return;
    if (r.savedOnly) { toast(`A minta Excel elmentve (${r.savedOnly}), de más néven, ezért nem töltöttem be: a program csak a(z) „${(S.import || {}).excelName}” nevű Excelt olvassa.`, 'warn', { timeout: 12000 }); return; }
    applyExcel(r, true);
    toast('A minta Excel elmentve és betöltve. Nyisd meg Excelben, írd át a saját partnereidre, mentsd, majd nyomd meg az Újratöltés gombot.', 'ok', { timeout: 9000 });
  } catch (e) { toast(e.message, 'err'); }
}

async function uploadExcel(file) {
  if (!/\.(xlsx|xlsm)$/i.test(file.name)) {
    toast('Csak .xlsx (vagy .xlsm) munkafüzet tölthető be. A régi .xls fájlt mentsd el Excelben .xlsx formátumban.', 'warn');
    return;
  }
  try {
    const r = await api('/api/excel/upload', file, { raw: true, headers: { 'X-Filename': encodeURIComponent(file.name) } });
    applyExcel(r);
    setTab('adatok');
  } catch (e) { toast(e.message, 'err'); }
}

function setupDrop() {
  let depth = 0;
  const ov = $('#dropOverlay');
  const hasFiles = e => e.dataTransfer && Array.from(e.dataTransfer.types || []).includes('Files');
  window.addEventListener('dragenter', e => { if (!hasFiles(e)) return; e.preventDefault(); depth++; ov.classList.add('on'); });
  window.addEventListener('dragleave', e => { if (!hasFiles(e)) return; depth = Math.max(0, depth - 1); if (!depth) ov.classList.remove('on'); });
  window.addEventListener('dragover', e => { e.preventDefault(); });
  window.addEventListener('drop', e => {
    e.preventDefault();
    depth = 0;
    ov.classList.remove('on');
    const files = Array.from((e.dataTransfer && e.dataTransfer.files) || []);
    const tpls = files.filter(f => /\.(html?|zip)$/i.test(f.name));
    const json = files.find(f => /\.json$/i.test(f.name));
    const txt = files.find(f => /\.txt$/i.test(f.name));
    if (tpls.length) uploadTemplates(tpls);
    else if (json) b2bImportFile(json);
    else if (txt) b2bImportSourcesText(txt);
    else if (files[0]) uploadExcel(files[0]);
  });
}

/* ------------------------------------------------------------------ 2. Tartalom */

// A Tartalom fülön a változók listája (a betöltött partnerlista saját mezőivel együtt).
function tokenNote() {
  const b2b = S.excel && S.excel.source === 'b2b';
  return h('div', { class: 'note cream tok-note' }, icon('users'),
    h('div', null, h('b', { text: 'Partnerenként cserélődő változók: ' }),
      S.tokens.slice(0, 7).concat((S.excel && S.excel.tokens) || []).map((t, i) => [i ? ' ' : '', h('span', { class: 'tok', title: t.desc, text: t.token })]),
      '. Pl. a megszólítás „Kedves {nev}!” – a {nev} helyére minden levélben a partner neve kerül.',
      b2b ? ' A B2B partnertörzsből a {nev} és a {ceg} is a partner neve, a {terulet} a partner megyéje.' : ''));
}

function refreshTokenNote() {
  const n = $('.tok-note');
  if (n) n.replaceWith(tokenNote());
}

function renderContent() {
  const p = pane('tartalom');
  const open = new Set(ls('groups') || ['alap', 'level']);
  p.replaceChildren(
    secHead('02 / Közös tartalom', 'Ami minden partnernél ugyanaz',
      'A mezők a hírlevél blokkjait követik, fentről lefelé. A jobb oldali előnézet gépelés közben frissül.'),
    tokenNote());
  for (const g of S.groups) {
    const fields = S.fields.filter(f => f.group === g.id);
    const card = h('div', { class: 'card group' + (open.has(g.id) ? ' open' : ''), dataset: { group: g.id } });
    const head = h('div', { class: 'group-head', onclick: () => {
      card.classList.toggle('open');
      autosizeAll(card);
      const o = new Set(ls('groups') || ['alap', 'level']);
      card.classList.contains('open') ? o.add(g.id) : o.delete(g.id);
      ls('groups', Array.from(o));
    } },
      h('div', { class: 'group-ic' }, icon(g.icon, 20)),
      h('div', { style: { minWidth: 0 } },
        h('div', { class: 'group-title' }, g.title, g.poll ? h('span', { class: 'pill dark', text: 'kérdéses sablonban' }) : null),
        h('div', { class: 'group-desc', text: g.desc })),
      h('div', { class: 'group-meta' }, h('span', { class: 'gstat' }), icon('chev', 18)));
    head.lastChild.lastChild.classList.add('chev');
    const body = h('div', { class: 'group-body' },
      g.poll ? h('div', { class: 'inactive-note hidden', text: 'A kiválasztott sablon ezt a blokkot nem használja, a mezők értéke megmarad.' }) : null,
      h('div', { class: 'fields' }, fields.map(fieldRow)));
    card.append(head, body);
    p.append(card);
  }
  updateTemplateDependent();
  updateFieldIssues();
}

function autosize(el) {
  if (!el.offsetParent) return;
  el.style.height = 'auto';
  el.style.height = (el.scrollHeight + 2) + 'px';
}

function autosizeAll(root) {
  requestAnimationFrame(() => $$('textarea.inp', root).forEach(autosize));
}

function fieldRow(f) {
  const val = S.state.content[f.key] || '';
  const counter = h('span', { class: 'counter' });
  const isArea = f.kind === 'textarea';
  const input = h(isArea ? 'textarea' : 'input', {
    class: 'inp', id: 'f-' + f.key, value: val, placeholder: f.placeholder || '', spellcheck: !(f.kind === 'url' || f.kind === 'image'),
    rows: isArea ? 2 : null, type: isArea ? null : 'text', dataset: { key: f.key },
  });
  const wrap = h('div', { class: 'field' + (f.half ? ' half' : ''), dataset: { key: f.key } });
  let thumb = null;
  if (f.kind === 'image') thumb = h('div', { class: 'thumb' + (f.key === 'cover.image' ? ' wide' : '') });
  const updThumb = () => { if (thumb) thumb.style.backgroundImage = input.value.trim() ? `url("${resolveUrl(input.value).replace(/"/g, '%22')}")` : ''; };
  const updCounter = () => {
    const n = Array.from(f.tokens ? expandPreview(input.value) : input.value).length;
    if (f.soft) {
      counter.textContent = `${n} / ${f.soft}`;
      counter.classList.toggle('over', n > f.soft || (f.min && n > 0 && n < f.min));
    } else counter.textContent = '';
  };
  const grow = () => { if (isArea) autosize(input); };
  const thumbSoon = debounce(updThumb, 500);
  input.addEventListener('input', () => {
    S.state.content[f.key] = input.value;
    updCounter(); grow(); thumbSoon();
    syncSoon();
  });
  if (f.tokens) {
    input.addEventListener('focus', () => showTokenBar(wrap, input));
    input._recount = updCounter; // a partner váltásakor újraszámoljuk (a változók hossza más)
  }
  wrap.append(
    h('div', { class: 'field-top' }, h('label', { for: input.id }, f.label, f.required ? h('span', { class: 'req', text: '*' }) : null), counter),
    thumb ? h('div', { class: 'img-field' }, input, thumb) : input,
    f.help ? h('div', { class: 'help', text: f.help }) : null,
    h('div', { class: 'msg' }));
  updCounter(); updThumb();
  requestAnimationFrame(grow);
  return wrap;
}

// A beszúrható változók: a beépítettek és a betöltött partnerlista saját mezői (Excel-oszlopok, B2B).
function allTokens() {
  return S.tokens.concat((S.excel && S.excel.tokens) || []);
}

function showTokenBar(wrap, input) {
  $$('.tokenbar').forEach(t => { if (t.parentElement !== wrap) t.remove(); });
  if ($('.tokenbar', wrap)) return;
  const isUrl = input.dataset.key && /url|Pattern$/i.test(input.dataset.key);
  const tokens = allTokens().filter(t => t.token !== '{assets}' || isUrl);
  const bar = h('div', { class: 'tokenbar' }, h('span', { class: 't-label', text: 'Változó beszúrása' }),
    tokens.map(t => h('button', { class: 'chip', type: 'button', title: t.desc, text: t.token,
      onmousedown: e => e.preventDefault(),
      onclick: () => {
        const s = input.selectionStart ?? input.value.length, e2 = input.selectionEnd ?? s;
        input.setRangeText(t.token, s, e2, 'end');
        input.dispatchEvent(new Event('input'));
        input.focus();
      } })));
  const msg = $('.msg', wrap);
  wrap.insertBefore(bar, msg);
  input.addEventListener('blur', () => setTimeout(() => { if (document.activeElement !== input) bar.remove(); }, 150), { once: true });
}

function updateTemplateDependent() {
  const hasPoll = !!curTpl().hasPoll;
  $$('.group').forEach(card => {
    const g = S.groups.find(x => x.id === card.dataset.group);
    const inactive = !!(g && g.poll && !hasPoll);
    card.classList.toggle('inactive', inactive);
    const note = $('.inactive-note', card);
    if (note) note.classList.toggle('hidden', !inactive);
  });
  for (const f of S.fields) {
    const el = $(`.field[data-key="${f.key}"]`, pane('tartalom'));
    if (el) el.classList.toggle('dim', !!(f.poll && !hasPoll));
  }
}

function updateFieldIssues() {
  const byKey = new Map();
  for (const i of contentIssues('content')) {
    if (i.level === 'info') continue;
    const cur = byKey.get(i.key);
    if (!cur || (i.level === 'error' && cur.level !== 'error')) byKey.set(i.key, i);
  }
  $$('.field[data-key]', pane('tartalom')).forEach(el => {
    const i = byKey.get(el.dataset.key);
    el.classList.toggle('has-err', !!(i && i.level === 'error'));
    el.classList.toggle('has-warn', !!(i && i.level === 'warn'));
    const m = $('.msg', el);
    if (m) m.textContent = i ? i.message : '';
  });
  $$('.group', pane('tartalom')).forEach(card => {
    const keys = new Set(S.fields.filter(f => f.group === card.dataset.group).map(f => f.key));
    let e = 0, w = 0;
    for (const i of contentIssues('content')) {
      if (!keys.has(i.key)) continue;
      if (i.level === 'error') e++; else if (i.level === 'warn') w++;
    }
    const st = $('.gstat', card);
    st.replaceChildren(e ? h('span', { class: 'pill err', text: e + ' hiba' }) : null, !e && w ? h('span', { class: 'pill warn', text: w + ' figyelmeztetés' }) : null);
  });
}

async function exportContent() {
  try {
    await syncNow();
    const r = await api('/api/content/export');
    if (!r.cancelled) toast('A tartalom elmentve.', 'ok', { url: r.path });
  } catch (e) { toast(e.message, 'err'); }
}

function applyState(r) {
  S.state = r.state;
  if (!S.state.feed) S.state.feed = {};
  setIssues(r.issues);
  renderContent(); renderProducts(); renderTemplateSeg();
  if (S.tab === 'generalas') renderGenerate();
  refreshPreview();
}

async function importContent() {
  try {
    const r = await api('/api/content/import');
    if (r.cancelled) return;
    applyState(r);
    toast('A tartalom betöltve.', 'ok', { url: r.path });
  } catch (e) { toast(e.message, 'err'); }
}

async function resetContent() {
  if (!await confirmBox('Visszaállítás a mintára', 'A közös tartalom minden mezője a tervezői mintaszövegre áll vissza. A termékek és a partnerlista nem változik.', 'Visszaállítás', true)) return;
  try { applyState(await api('/api/content/defaults')); toast('A közös tartalom visszaállt a mintára.', 'ok'); } catch (e) { toast(e.message, 'err'); }
}

/* ------------------------------------------------------------------ 3. Termékek */

const MAX_PRODUCT_CARDS = 60;
const MAX_PARTNER_ROWS = 300;
const MAX_PREVIEW_OPTIONS = 1000;

const PFIELDS = [
  { k: 'code', label: 'Cikkszám', half: true, req: true },
  { k: 'price', label: 'Ár', half: true, ph: 'pl. 3 090 Ft' },
  { k: 'name', label: 'Cikknév', soft: 26, req: true },
  { k: 'desc', label: 'Rövid leírás', soft: 30 },
  { k: 'deal', label: 'Akció (opcionális)', soft: 18, half: true },
  { k: 'cta', label: 'Gombfelirat', half: true },
  { k: 'image', label: 'Cikk kép link', url: true },
  { k: 'url', label: 'Gomb link', url: true },
];

function renderProducts() {
  const p = pane('termekek');
  const list = S.state.products;
  const sel = list.filter(x => x.on).length;
  const exN = S.excel && S.excel.products ? S.excel.products.length : 0;
  p.replaceChildren(
    secHead('03 / Ajánlat', 'Termékek',
      'Az Excel Termékek munkalapjáról. Itt ki-be kapcsolhatod, sorba rendezheted és javíthatod őket – a változás csak a hírlevélre hat, az Excel-fájlt nem módosítja.'),
    h('div', { class: 'toolbar' },
      h('div', { class: 'count-big' }, `${sel} termék a levélben `, h('small', { text: `/ ${list.length} betöltve` })),
      h('div', { class: 'grow' }),
      exN ? h('button', { class: 'btn btn-outline btn-sm', onclick: e => busy(e.currentTarget, reloadProducts) }, icon('refresh', 16), 'Újratöltés az Excelből') : null,
      list.length ? h('button', { class: 'btn btn-ghost btn-sm btn-danger', title: 'Az összes termék törlése a hírlevélből', onclick: () => resetData(true, false) }, icon('trash', 16), 'Összes törlése') : null,
      h('button', { class: 'btn btn-primary btn-sm', onclick: openPicker }, icon('plus', 16), 'Új termék')),
    h('div', { id: 'pGeneral' }),
    list.length > MAX_PRODUCT_CARDS ? h('div', { class: 'note warn' }, icon('alert'), h('div', null, `${list.length} termék van betöltve – ez nem hírlevél-terméklista (egy levélbe legfeljebb 48 kerülhet). Csak az első ${MAX_PRODUCT_CARDS} látszik. `,
      h('button', { class: 'linkbtn', text: 'Összes törlése', onclick: () => resetData(true, false) }))) : null,
    h('div', { id: 'plist' }, list.length ? list.slice(0, MAX_PRODUCT_CARDS).map(productCard) : h('div', { class: 'card empty' }, 'Még nincs termék. Tölts be Excelt Termékek munkalappal, vagy az „Új termék” gombbal keress a cikktörzsben.')));
  updateProductIssues();
  renderSteps();
  lookupImages();
}

function productPos(i) {
  let n = 0;
  for (let k = 0; k <= i; k++) if (S.state.products[k].on) n++;
  return n;
}

function productCard(pr, i) {
  const c = S.state.content;
  const thumbBox = h('div', { class: 'p-img' });
  const updThumb = () => {
    const url = resolveUrl(pr.image || c['offer.imagePattern'], pr.code);
    thumbBox.replaceChildren(pr.on ? h('span', { class: 'p-pos', text: productPos(i) + '.' }) : null);
    if (!url) { thumbBox.append(h('div', { class: 'ph', text: 'nincs kép' })); return; }
    const img = h('img', { src: url, alt: pr.name || '' });
    img.onerror = () => img.replaceWith(h('div', { class: 'ph' }, icon('image', 22), h('div', { text: 'a kép itt nem tölthető be' })));
    thumbBox.append(img);
  };
  const thumbSoon = debounce(updThumb, 500);
  const card = h('div', { class: 'card pcard' + (pr.on ? '' : ' off'), dataset: { i } });
  const fields = PFIELDS.map(f => {
    const inp = h('input', { class: 'inp', type: 'text', value: pr[f.k] || '', dataset: { k: f.k }, spellcheck: !f.url,
      placeholder: f.k === 'cta' ? (c['offer.cta'] || '') : f.k === 'image' ? resolveUrl(c['offer.imagePattern'], pr.code || '…') : f.k === 'url' ? resolveUrl(c['offer.urlPattern'], pr.code || '…') : (f.ph || '') });
    const counter = h('span', { class: 'counter' });
    const upd = () => {
      if (!f.soft) return;
      const n = Array.from(inp.value.trim()).length;
      counter.textContent = `${n} / ${f.soft}`;
      counter.classList.toggle('over', n > f.soft);
    };
    inp.addEventListener('input', () => {
      pr[f.k] = inp.value;
      upd();
      if (f.k === 'image' || f.k === 'code') thumbSoon();
      syncSoon();
    });
    if (f.k === 'price') inp.addEventListener('blur', () => { const v = formatPrice(inp.value); if (v !== inp.value) { inp.value = v; pr.price = v; syncSoon(0); } });
    if (f.k === 'code') inp.addEventListener('input', () => { if (pr.images) { delete pr.images; const fn = swRender.get(pr); if (fn) fn(); } lookupSoon(); });
    if (f.k === 'image') inp.addEventListener('input', () => { const fn = swRender.get(pr); if (fn) fn(); });
    upd();
    return h('div', { class: 'field' + (f.half ? ' half' : ''), dataset: { k: f.k } },
      h('div', { class: 'field-top' }, h('label', null, f.label, f.req ? h('span', { class: 'req', text: '*' }) : null), counter),
      inp, f.k === 'image' ? imageSwitcher(pr, inp, updThumb) : null, h('div', { class: 'msg' }));
  });
  const toggle = h('input', { type: 'checkbox', checked: pr.on, onchange: e => { pr.on = e.target.checked; renderProducts(); syncSoon(0); } });
  card.append(
    h('div', { class: 'p-side' },
      thumbBox,
      h('label', { class: 'p-toggle' }, toggle, 'A levélben'),
      h('div', { class: 'p-order' },
        h('button', { class: 'btn btn-ghost btn-sm', title: 'Feljebb', disabled: i === 0, onclick: () => moveProduct(i, -1) }, icon('up', 16)),
        h('button', { class: 'btn btn-ghost btn-sm', title: 'Lejjebb', disabled: i === S.state.products.length - 1, onclick: () => moveProduct(i, 1) }, icon('down', 16)))),
    h('div', { class: 'fields' }, fields),
    h('button', { class: 'btn btn-ghost btn-sm btn-danger del', title: 'Termék törlése', onclick: () => removeProduct(i) }, icon('trash', 16)));
  updThumb();
  return card;
}

function moveProduct(i, d) {
  const L = S.state.products;
  const j = i + d;
  if (j < 0 || j >= L.length) return;
  [L[i], L[j]] = [L[j], L[i]];
  renderProducts();
  syncSoon(0);
}

function removeProduct(i) {
  const [removed] = S.state.products.splice(i, 1);
  renderProducts();
  syncSoon(0);
  toast(`Törölve: ${removed.name || removed.code || 'termék'}`, 'info', { actions: [{ label: 'Visszavonás', fn: () => { S.state.products.splice(i, 0, removed); renderProducts(); syncSoon(0); } }] });
}

function addEmptyProduct() {
  S.state.products.push({ on: true, code: '', name: '', desc: '', price: '', deal: '', image: '', url: '', alt: '', cta: '' });
  renderProducts();
  syncSoon(0);
  const last = $('#plist').lastElementChild;
  if (last) { last.scrollIntoView({ block: 'center' }); $('input.inp', last).focus(); }
}

/* ------------------------------------------------------------------ cikktörzs (termékfeed) */

const IMG_LABEL = { thumb: 'Bélyegkép', code: 'Cikkkép', small: 'Kis kép', large: 'Nagy kép' };
const IMG_ORDER = {
  code: ['code', 'large', 'small', 'thumb'], large: ['large', 'code', 'small', 'thumb'],
  small: ['small', 'code', 'large', 'thumb'], thumb: ['thumb', 'small', 'code', 'large'],
};

function imgLabel(id) {
  const m = /^gallery(\d+)$/.exec(id || '');
  return m ? 'Galéria ' + m[1] : (IMG_LABEL[id] || id);
}

function imgLevel(id) {
  return id === 'code' || id === 'large'
    ? 'ennek a cikknek (változatnak) a saját képe'
    : 'a főtermék közös képe – szín- vagy méretváltozatnál eltérhet ettől a cikktől';
}

function isWebp(u) { return /\.webp$/i.test(String(u || '').split('?')[0]); }

// Az előnyben részesített kitöltött kép; WEBP-t (az Outlook nem mutatja) csak végső esetben.
function pickImg(images, pref) {
  const list = images || [];
  for (const allowWebp of [false, true]) {
    for (const id of IMG_ORDER[pref] || IMG_ORDER.code) {
      const im = list.find(x => x.id === id && (allowWebp || !isWebp(x.url)));
      if (im) return im.url;
    }
    const g = list.find(x => allowWebp || !isWebp(x.url));
    if (g) return g.url;
  }
  return '';
}

function codeKey(c) { return norm(c).replace(/-/g, '').trim(); }

function fmtFt(n) { return n > 0 ? formatPrice(String(n)) : ''; }

function fmtMB(b) { return (b / 1e6).toLocaleString('hu-HU', { maximumFractionDigits: 1, minimumFractionDigits: 1 }) + ' MB'; }

function feedOpts() {
  if (!S.state.feed) S.state.feed = {};
  return S.state.feed;
}

// A cikktörzs állapotának figyelése, amíg betöltés fut; a változásra feliratkozók értesülnek.
const feedListeners = new Set();
let feedPollTimer = null;

function setFeedStatus(st) {
  const wasReady = S.feed && S.feed.ready && S.feed.dataTime === st.dataTime && S.feed.count === st.count;
  S.feed = st;
  for (const fn of feedListeners) fn(st, !wasReady && st.ready);
  if (st.ready && !wasReady) { lookupDone.clear(); lookupImages(); }
  clearTimeout(feedPollTimer);
  if (st.loading) feedPollTimer = setTimeout(pollFeed, 450);
}

async function pollFeed() {
  try { setFeedStatus(await api('/api/feed/status')); } catch (e) { feedPollTimer = setTimeout(pollFeed, 2000); }
}

async function refreshFeed(force, url) {
  try { setFeedStatus(await api('/api/feed/refresh', { force: !!force, url: url || '' })); } catch (e) { toast(e.message, 'err'); }
}

// Az Excelből vagy korábbról származó termékek képváltozatai a cikktörzsből (cikkszám alapján).
const lookupDone = new Set();
let lookupBusy = false;

async function lookupImages() {
  if (lookupBusy || !S.feed || !S.feed.ready) return;
  const want = S.state.products.filter(p => p.code && !(p.images && p.images.length) && !lookupDone.has(codeKey(p.code)));
  if (!want.length) return;
  lookupBusy = true;
  try {
    const codes = [...new Set(want.map(p => p.code.trim()))];
    const r = await api('/api/feed/images', { codes });
    if (!r.ready) return;
    codes.forEach(c => { if (!r.images[c]) lookupDone.add(codeKey(c)); }); // a nem találtakat nem kérdezi újra
    let changed = false;
    for (const p of want) {
      const imgs = r.images[p.code.trim()];
      if (!imgs || !imgs.length) continue;
      p.images = imgs;
      // kézzel felvett termék üres képmezője: az előnyben részesített kép (az Excel-sorokét nem írja felül)
      if (!String(p.image || '').trim() && !p.row) p.image = pickImg(imgs, feedOpts().image);
      changed = true;
      const fn = swRender.get(p);
      if (fn) fn();
    }
    if (changed) syncSoon();
  } catch (e) { /* csendben: a képváltó enélkül is működik */ } finally { lookupBusy = false; }
}
const lookupSoon = debounce(lookupImages, 700);

// Egy (pl. kézzel felvett) termék üres mezőinek kitöltése a cikktörzsből.
async function fillFromFeed(pr) {
  try {
    const r = await api('/api/feed/products', { codes: [String(pr.code || '').trim()], options: feedOpts() });
    const f = r.products[0];
    if (!f) { toast('Ez a cikkszám nincs a cikktörzsben.', 'warn'); return; }
    const keys = ['name', 'desc', 'price', 'deal', 'image', 'url', 'alt'];
    let take = keys.filter(k => !String(pr[k] || '').trim() && f[k]);
    if (!take.filter(k => k !== 'alt').length) {
      if (!await confirmBox('Kitöltés a cikktörzsből', 'Minden mező ki van töltve. Felülírod őket a cikktörzs adataival (név, rövid leírás, ár, akció, kép, gomb link)?', 'Felülírás')) return;
      take = keys.filter(k => f[k]);
    }
    for (const k of take) pr[k] = f[k];
    pr.images = f.images;
    renderProducts();
    syncSoon(0);
    toast(`${take.filter(k => k !== 'alt').length} mező kitöltve a cikktörzsből – utána is szabadon átírható.`, 'ok');
  } catch (e) { toast(e.message, 'err'); }
}

// Képváltó a termékkártyán: a cikktörzsben kitöltött képméretek közül lehet választani.
const swRender = new WeakMap();

function imageSwitcher(pr, inp, onPick) {
  const box = h('div', { class: 'imgsw' });
  let expanded = false;
  const render = () => {
    const imgs = pr.images || [];
    if (inp.value !== (pr.image || '')) { inp.value = pr.image || ''; onPick(); }
    box.replaceChildren();
    if (!imgs.length) return;
    const gal = imgs.filter(x => /^gallery/.test(x.id));
    const shown = expanded ? imgs : imgs.filter(x => !/^gallery/.test(x.id)).concat(gal.slice(0, 3));
    const cur = String(pr.image || '').trim();
    box.append(h('span', { class: 'sw-label', text: 'Képméret:' }),
      shown.map(im => {
        const dims = h('small', { text: '' });
        const pic = h('img', { src: im.url, alt: '', loading: 'lazy' });
        const webp = isWebp(im.url);
        pic.onload = () => { dims.textContent = pic.naturalWidth + '×' + pic.naturalHeight + (webp ? ' · WEBP' : ''); };
        pic.onerror = () => { b.classList.add('broken'); dims.textContent = 'nem tölthető be'; };
        const b = h('button', { type: 'button', class: 'sw' + (cur === im.url ? ' on' : '') + (webp ? ' webp' : ''),
          title: imgLabel(im.id) + ' – ' + imgLevel(im.id) + (webp ? '\nWEBP: az Outlook asztali változata nem jeleníti meg' : '') + '\n' + im.url,
          onclick: () => { pr.image = im.url; inp.value = im.url; onPick(); render(); syncSoon(0); } },
          pic, h('span', null, h('b', { text: imgLabel(im.id) }), dims));
        return b;
      }),
      !expanded && gal.length > 3 ? h('button', { type: 'button', class: 'sw more', text: `+${gal.length - 3} galériakép`, onclick: () => { expanded = true; render(); } }) : null,
      h('button', { type: 'button', class: 'sw fill', title: 'A termék üres mezőit (név, leírás, ár, akció, kép, link) kitölti a cikktörzs adataival', onclick: () => fillFromFeed(pr) },
        icon('download', 14), 'Kitöltés a cikktörzsből'));
  };
  swRender.set(pr, render);
  render();
  return box;
}

// „Új termék”: keresés a cikktörzsben és átvétel a meglévő mezőkbe.
function openPicker() {
  const o = feedOpts();
  let results = [], sel = 0, added = 0, seq = 0, lastQ = null;
  const statusBox = h('div', { class: 'pk-status' });
  const list = h('div', { class: 'pk-list', role: 'listbox' });
  const search = h('input', { class: 'inp pk-q', type: 'search', placeholder: 'Cikkszám vagy név, pl. 10000-327 vagy wizard crab', spellcheck: false, autocomplete: 'off' });
  const addedInfo = h('div', { class: 'pk-added' });
  const urlRow = h('div', { class: 'pk-url', hidden: true });

  const seg = (label, key, opts, def) => h('div', { class: 'pk-opt' }, h('span', { class: 'label-caps', text: label }),
    h('div', { class: 'seg' }, opts.map(([v, t, tip]) => h('button', { type: 'button', class: (o[key] || def) === v ? 'on' : '', title: tip || null, text: t,
      onclick: e => { o[key] = v; $$('button', e.currentTarget.parentNode).forEach(b => b.classList.toggle('on', b === e.currentTarget)); syncSoon(); drawList(); } }))));
  const optsBar = h('div', { class: 'pk-opts' },
    seg('Ár', 'price', [['retail', 'Kisker bruttó', 'Kisker_brutto (akció esetén az akciós ár)'], ['wholesale', 'Nagyker nettó', 'Nagyker_netto „+ áfa” jelöléssel'], ['none', 'Ne töltse ki']], 'retail'),
    seg('Kép', 'image', [['code', 'Cikkkép', 'a változat saját képe (ha nincs: nagy, kis, bélyegkép)'], ['large', 'Nagy'], ['small', 'Kis'], ['thumb', 'Bélyeg']], 'code'),
    seg('Gomb link', 'link', [['webshop', 'Webshop oldal', 'a cikktörzsben megadott termékoldal'], ['pattern', 'Haladó minta', 'a Tartalom › Haladó beállítások termékoldal-mintája']], 'webshop'));

  const done = () => {
    feedListeners.delete(onFeed);
    document.removeEventListener('keydown', key);
    bg.remove();
    if (added) {
      const last = $('#plist') && $('#plist').lastElementChild;
      if (last) last.scrollIntoView({ block: 'center' });
    }
  };
  const key = e => { if (e.key === 'Escape') done(); };

  function statusLine(st) {
    const parts = [];
    if (st.loading && !st.ready) {
      const pct = st.total > 0 ? Math.min(100, Math.round(st.bytes / st.total * 100)) : null;
      parts.push(h('div', { class: 'pk-load' }, h('span', { class: 'spin dark' }),
        h('span', { text: st.phase === 'cache' ? 'A gépre mentett cikktörzs betöltése…' : `A cikktörzs letöltése… ${fmtMB(st.bytes)}${st.total > 0 ? ' / ' + fmtMB(st.total) : ''}` })),
        pct != null ? h('div', { class: 'bar' }, h('i', { style: { width: pct + '%' } })) : null);
    } else if (!st.ready) {
      parts.push(h('div', { class: 'note err' }, icon('error'), h('div', null, h('b', { text: 'A cikktörzs nem érhető el. ' }), st.lastError || 'Ismeretlen hiba.',
        h('div', { class: 'help', text: 'Ellenőrizd az internetkapcsolatot, vagy a lenti „Cikktörzs címe” beállítást. Addig a terméket kézzel is felveheted.' }))));
    } else {
      parts.push(h('div', { class: 'pk-ok' },
        h('span', { class: 'dot' + (st.lastError ? ' warn' : '') }),
        h('span', null, h('b', { text: st.count.toLocaleString('hu-HU') + ' cikk' }), ' a cikktörzsben',
          st.dataTime ? ` · letöltve: ${fmtTime(st.dataTime)}` : '',
          st.loading ? ' · frissítés keresése…' : ''),
        st.loading ? h('span', { class: 'spin dark' }) : null,
        h('button', { class: 'btn btn-ghost btn-sm', title: 'A cikktörzs újraletöltése most', disabled: st.loading, onclick: () => refreshFeed(true) }, icon('refresh', 15), 'Frissítés')));
      if (st.lastError) parts.push(h('div', { class: 'note warn' }, icon('alert'), h('div', { text: 'A frissítés nem sikerült (' + st.lastError + '). A legutóbb letöltött cikktörzsből keresel.' })));
    }
    statusBox.replaceChildren(...parts);
  }

  function onFeed(st, becameReady) {
    statusLine(st);
    if (becameReady) { lastQ = null; doSearch(); }
  }

  const priceBlock = p => {
    const rows = [];
    const row = (label, list, sale, suffix) => {
      if (!list && !sale) return null;
      const on = sale > 0 && (sale < list || !list);
      return h('div', { class: 'pk-price' }, h('span', { class: 'lbl', text: label }),
        h('b', { text: fmtFt(on ? sale : list) + (suffix || '') }),
        on && list ? h('s', { text: fmtFt(list) }) : null);
    };
    rows.push(row('kisker', p.retail, p.retailSale), row('nagyker', p.wholesale, p.wholesaleSale, ' + áfa'));
    return h('div', { class: 'pk-prices' }, rows);
  };

  function drawList() {
    const q = search.value.trim();
    if (!S.feed || !S.feed.ready) { list.replaceChildren(); return; }
    if (!q) {
      list.replaceChildren(h('div', { class: 'pk-empty' }, icon('search', 28),
        h('div', { text: 'Kezdd el beírni a cikkszámot vagy a termék nevét.' }),
        h('div', { class: 'help', text: 'Kötőjel nélkül is megtalálja (10000327), az ékezet és a kis-nagybetű mindegy. Enterrel a kijelölt találat bekerül a termékek közé, így egymás után több cikkszámot is felvehetsz.' })));
      return;
    }
    if (!results.length) { list.replaceChildren(h('div', { class: 'pk-empty' }, h('div', { text: `Nincs találat erre: „${q}”.` }))); return; }
    list.replaceChildren(results.map((p, i) => {
      const thumb = pickImg(p.images, 'thumb');
      const img = thumb ? h('img', { src: thumb, alt: '', loading: 'lazy' }) : h('div', { class: 'ph' }, icon('image', 18));
      if (thumb) img.onerror = () => img.replaceWith(h('div', { class: 'ph' }, icon('image', 18)));
      const stock = p.stock === 1 ? h('span', { class: 'pill ok', text: 'készleten' }) : p.stock === 0 ? h('span', { class: 'pill err', text: 'nincs készleten' }) : h('span', { class: 'pill muted', text: 'készlet: ?' });
      const sale = (p.retailSale > 0 && p.retailSale < p.retail) || (p.wholesaleSale > 0 && p.wholesaleSale < p.wholesale);
      const row = h('div', { class: 'pk-row' + (i === sel ? ' sel' : ''), role: 'option', onmouseenter: () => { sel = i; mark(); }, ondblclick: () => addCodes([p.code], row) },
        h('div', { class: 'pk-img' }, img),
        h('div', { class: 'pk-main' },
          h('div', { class: 'pk-code' }, h('b', { text: p.code }), p.added ? h('span', { class: 'pill dark', text: 'már a listában' }) : null),
          h('div', { class: 'pk-name', text: p.name }),
          h('div', { class: 'pk-meta', text: [p.brand, [p.category, p.subcategory].filter(Boolean).join(' › ')].filter(Boolean).join(' · ') }),
          h('div', { class: 'pk-tags' }, stock, sale ? h('span', { class: 'pill warn', text: 'akciós' }) : null,
            p.badge ? h('span', { class: 'pill ' + (p.badge === 'EFTTEX díjas' ? 'ok' : 'warn'), text: p.badge }) : null,
            h('span', { class: 'pk-imgs', title: (p.images || []).map(x => imgLabel(x.id)).join(', ') || 'nincs kép', text: (p.images || []).length + ' kép' }))),
        priceBlock(p),
        h('button', { class: 'btn btn-primary btn-sm', onclick: e => addCodes([p.code], e.currentTarget) }, icon('plus', 15), 'Hozzáadás'));
      return row;
    }), moreNote);
  }
  let moreNote = null;

  function mark() {
    $$('.pk-row', list).forEach((r, i) => r.classList.toggle('sel', i === sel));
  }

  async function doSearch() {
    const q = search.value.trim();
    if (q === lastQ) return;
    lastQ = q;
    const my = ++seq;
    if (!q || !S.feed || !S.feed.ready) { results = []; moreNote = null; drawList(); return; }
    try {
      const r = await api('/api/feed/search', { q, limit: 60 });
      if (my !== seq) return;
      results = r.results || [];
      sel = 0;
      moreNote = r.more ? h('div', { class: 'pk-more', text: 'Csak az első 60 találat látszik – pontosítsd a keresést.' }) : null;
      drawList();
    } catch (e) { toast(e.message, 'err'); }
  }
  const searchSoon = debounce(doSearch, 140);

  async function addCodes(codes, btn) {
    try {
      const r = await api('/api/feed/products', { codes, options: feedOpts() });
      S.state.products.push(...r.products);
      added += r.products.length;
      for (const p of results) if (codes.includes(p.code)) p.added = true;
      renderProducts();
      syncSoon(0);
      drawList();
      addedInfo.replaceChildren(icon('check', 16), h('span', { text: `${added} termék hozzáadva – a termékek listájában bármelyik mezőjük átírható.` }));
      addedInfo.classList.add('on');
      if (btn && btn.closest) { const row = btn.closest('.pk-row'); if (row) flash(row); }
    } catch (e) { toast(e.message, 'err'); }
  }

  search.addEventListener('input', searchSoon);
  search.addEventListener('keydown', async e => {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (!results.length) return;
      sel = (sel + (e.key === 'ArrowDown' ? 1 : -1) + results.length) % results.length;
      mark();
      const r = $$('.pk-row', list)[sel];
      if (r) r.scrollIntoView({ block: 'nearest' });
    } else if (e.key === 'Enter') {
      e.preventDefault();
      await doSearch();
      if (results[sel]) { await addCodes([results[sel].code]); search.select(); }
    }
  });

  const urlInp = h('input', { class: 'inp', type: 'url', value: o.url || '', placeholder: S.feedURL || '', spellcheck: false });
  urlRow.append(h('label', { class: 'label-caps', text: 'Cikktörzs címe (XML)' }),
    h('div', { class: 'pk-url-row' }, urlInp,
      h('button', { class: 'btn btn-outline btn-sm', text: 'Mentés és letöltés', onclick: () => {
        o.url = urlInp.value.trim();
        syncSoon(0);
        refreshFeed(true, o.url);
      } }),
      h('button', { class: 'btn btn-ghost btn-sm', text: 'Alapértelmezett', onclick: () => { urlInp.value = ''; o.url = ''; syncSoon(0); refreshFeed(true, ''); } })),
    h('div', { class: 'help', text: 'Üresen az Energofish nagyker termékfeedje. A letöltött cikktörzs a gépre mentődik; legközelebb csak akkor töltődik le újra, ha a szerveren változott.' }));

  const bg = h('div', { class: 'modal-bg', onclick: e => { if (e.target === bg) done(); } },
    h('div', { class: 'modal picker', role: 'dialog', 'aria-label': 'Termék hozzáadása a cikktörzsből' },
      h('div', { class: 'mh' }, h('span', { text: 'Termék hozzáadása a cikktörzsből' }),
        h('button', { class: 'x', title: 'Bezárás', onclick: done }, icon('x', 18))),
      h('div', { class: 'pk-top' },
        statusBox,
        h('div', { class: 'pk-search' }, icon('search', 18), search),
        optsBar),
      list,
      urlRow,
      h('div', { class: 'mf pk-foot' }, addedInfo, h('div', { class: 'grow' }),
        h('button', { class: 'btn btn-ghost btn-sm', onclick: () => { urlRow.hidden = !urlRow.hidden; } }, icon('link', 15), 'Cikktörzs címe'),
        h('button', { class: 'btn btn-outline btn-sm', onclick: () => { done(); addEmptyProduct(); } }, icon('pen', 15), 'Üres termék kézzel'),
        h('button', { class: 'btn btn-primary', text: 'Kész', onclick: done }))));
  document.body.append(bg);
  document.addEventListener('keydown', key);
  feedListeners.add(onFeed);
  statusLine(S.feed || { loading: true, ready: false, bytes: 0, total: 0, phase: 'download' });
  drawList();
  search.focus();
  refreshFeed(false, o.url); // friss cikktörzs: csak akkor töltődik le, ha a szerveren változott
}

async function reloadProducts() {
  if (!await confirmBox('Termékek újratöltése', 'A termékek az Excel Termékek munkalapjáról töltődnek be újra, az itt végzett módosítások elvesznek. (Ha az Excelt azóta módosítottad, előbb az Adatok lépésben töltsd újra a fájlt.)', 'Újratöltés')) return;
  try {
    const r = await api('/api/excel/products');
    S.state.products = r.products;
    setIssues(r.issues);
    renderProducts(); refreshPreview();
    toast('A termékek újratöltve az Excelből.', 'ok');
  } catch (e) { toast(e.message, 'err'); }
}

function updateProductIssues() {
  const gen = $('#pGeneral');
  if (!gen) return;
  const iss = contentIssues('product');
  gen.replaceChildren(...iss.filter(i => i.index < 0).map(noteFor));
  $$('.pcard').forEach(card => {
    const idx = +card.dataset.i;
    $$('.field[data-k]', card).forEach(f => {
      const i = iss.find(x => x.index === idx && x.key === f.dataset.k && x.level === 'error') || iss.find(x => x.index === idx && x.key === f.dataset.k);
      f.classList.toggle('has-err', !!(i && i.level === 'error'));
      f.classList.toggle('has-warn', !!(i && i.level === 'warn'));
      $('.msg', f).textContent = i ? i.message.replace(/^\d+\. termék( \([^)]*\))?: /, '') : '';
    });
  });
}

/* ------------------------------------------------------------------ 4. Ellenőrzés */

function renderCheck() {
  const p = pane('ellenorzes');
  const c = S.issues.counts || {};
  const blocked = S.issues.blocked || 0;
  let cls = 'ok', ic = 'check', title = 'Minden rendben, mehet a generálás', sub = 'Nincs hiba és figyelmeztetés.';
  if (c.error) { cls = 'err'; ic = 'error'; title = `${c.error} hiba javítandó`; sub = 'A hibás közös mezők és termékek miatt a generálás nem indítható; a hibás partnersorok kimaradnak.'; }
  else if (c.warn) { cls = 'warn'; ic = 'alert'; title = `Generálható · ${c.warn} figyelmeztetés`; sub = 'A figyelmeztetések nem akadályozzák a generálást, de érdemes átnézni őket.'; }
  const groups = [
    { t: 'Közös tartalom', ic: 'pen', list: contentIssues('content') },
    { t: 'Termékek', ic: 'grid', list: contentIssues('product') },
    { t: 'Excel és partnerek', ic: 'users', list: S.issues.partners },
  ];
  p.replaceChildren(
    secHead('04 / Ellenőrzés', 'Minden rendben a kiküldéshez?',
      'A program folyamatosan ellenőrzi a kötelező mezőket, a hosszkorlátokat és a linkeket. A képek elérhetőségét és méretét külön, online ellenőrizheted.'),
    h('div', { class: 'status-banner ' + cls },
      h('div', { class: 'big' }, icon(ic, 28)),
      h('div', null, h('h3', { text: title }), h('p', { text: sub + (blocked ? ` Kimaradó partner: ${blocked}.` : '') }))),
    groups.map(g => issueGroup(g)),
    imageCard());
}

function issueGroup(g) {
  const order = { error: 0, warn: 1, info: 2 };
  const list = g.list.slice().sort((a, b) => order[a.level] - order[b.level]);
  const e = list.filter(i => i.level === 'error').length, w = list.filter(i => i.level === 'warn').length;
  const card = h('div', { class: 'card issue-group' },
    h('h4', null, icon(g.ic, 18), g.t, h('span', { style: { marginLeft: 'auto', display: 'flex', gap: '6px' } },
      e ? h('span', { class: 'pill err', text: e + ' hiba' }) : null,
      w ? h('span', { class: 'pill warn', text: w + ' figyelmeztetés' }) : null,
      !e && !w ? h('span', { class: 'pill ok', text: 'rendben' }) : null)));
  const LIMIT = 150;
  if (!list.length) card.append(h('div', { class: 'issue' }, h('span', { class: 'st ok' }, icon('check', 15)), h('div', { class: 'txt', text: 'Nincs megjegyzés.' })));
  for (const i of list.slice(0, LIMIT)) {
    const st = i.level === 'error' ? 'err' : i.level;
    card.append(h('div', { class: 'issue' },
      h('span', { class: 'st ' + st }, icon(i.level === 'error' ? 'error' : i.level === 'warn' ? 'alert' : 'info', 15)),
      h('div', { class: 'txt', text: i.message }),
      canJump(i) ? h('button', { class: 'btn btn-ghost btn-sm', onclick: () => jumpTo(i) }, 'Ugrás', icon('right', 14)) : null));
  }
  if (list.length > LIMIT) card.append(h('div', { class: 'more-row', text: `…és még ${list.length - LIMIT} megjegyzés.` }));
  return card;
}

function canJump(i) {
  return (i.scope === 'content' && i.key) || i.scope === 'product' || (i.scope === 'partner' && i.index >= 0) || i.scope === 'excel';
}

function flash(el) {
  el.animate([{ boxShadow: '0 0 0 4px rgba(241,163,43,.7)' }, { boxShadow: '0 0 0 0 rgba(241,163,43,0)' }], { duration: 1400, easing: 'ease-out' });
}

function jumpTo(i) {
  if (i.scope === 'content') {
    setTab('tartalom');
    const f = S.fields.find(f => f.key === i.key);
    if (f) {
      const card = $(`.group[data-group="${f.group}"]`);
      if (card) { card.classList.add('open'); autosizeAll(card); }
    }
    const inp = $(`#f-${CSS.escape(i.key)}`);
    if (inp) { setTimeout(() => { inp.scrollIntoView({ block: 'center' }); inp.focus(); flash(inp); }, 60); }
  } else if (i.scope === 'product') {
    setTab('termekek');
    const card = i.index >= 0 ? $(`.pcard[data-i="${i.index}"]`) : null;
    if (card) setTimeout(() => {
      card.scrollIntoView({ block: 'center' });
      const inp = $(`.field[data-k="${i.key}"] input`, card) || $('input.inp', card);
      if (inp) { inp.focus(); flash(inp); }
    }, 60);
  } else {
    setTab('adatok');
    if (i.scope === 'partner' && i.index >= 0) {
      S.search = '';
      renderData();
      setPreviewPartner(i.index);
      setTimeout(() => { const tr = $(`#ptBody tr[data-i="${i.index}"]`); if (tr) { tr.scrollIntoView({ block: 'center' }); flash(tr); } }, 80);
    }
  }
}

function imageCard() {
  const card = h('div', { class: 'card card-pad' },
    h('div', { class: 'card-title' }, icon('image'), 'Képek ellenőrzése (online)'),
    h('p', { class: 'card-sub', text: 'Letölti a borító-, termék-, portré- és képviselőképeket, és ellenőrzi, hogy elérhetők-e, és megfelelő-e a méretük (borító 1200×660, termék min. 260 px, JPG/PNG). Internetkapcsolat kell hozzá.' }),
    h('button', { class: 'btn btn-dark', dataset: { busy: 'Képek letöltése…' }, onclick: e => busy(e.currentTarget, checkImages) }, icon('image', 16), S.imgResults ? 'Újraellenőrzés' : 'Képek ellenőrzése'));
  if (S.imgResults) {
    const r = S.imgResults;
    const bad = r.filter(x => x.status === 'error').length, warn = r.filter(x => x.status === 'warn').length, skip = r.filter(x => x.status === 'skip').length;
    card.append(h('div', { class: 'note ' + (bad ? 'err' : warn ? 'warn' : 'ok'), style: { margin: '16px 0 8px' } }, icon(bad ? 'error' : warn ? 'alert' : 'check'),
      h('div', { text: `${r.length - skip} kép ellenőrizve: ${bad} hibás, ${warn} figyelmeztetés` + (skip ? `, ${skip} helyi kép kihagyva.` : '.') })));
    const rows = r.map(x => h('tr', null,
      h('td', { class: 'ti' }, h('div', { style: { backgroundImage: x.status !== 'error' && x.status !== 'skip' ? `url("${x.url.replace(/"/g, '%22')}")` : '' } })),
      h('td', null, h('div', { style: { fontWeight: 700 }, text: x.where }), h('div', { class: 'p-sub mono', text: x.url })),
      h('td', { class: 'dim', text: x.width ? `${x.width}×${x.height}` : '' }),
      h('td', { class: 'dim', text: x.bytes ? Math.round(x.bytes / 1024) + ' KB' : '' }),
      h('td', null, h('span', { class: 'pill ' + ({ ok: 'ok', warn: 'warn', error: 'err', skip: 'muted' }[x.status]), text: { ok: 'rendben', warn: 'figyelem', error: 'hiba', skip: 'kihagyva' }[x.status] }),
        h('div', { class: 'p-sub', text: x.message }))));
    card.append(h('div', { class: 'table-wrap', style: { maxHeight: '420px', border: '1px solid var(--line)', borderRadius: '10px' } }, h('table', { class: 'img-results' }, h('tbody', null, rows))));
  }
  return card;
}

async function checkImages() {
  try {
    await syncNow();
    const r = await api('/api/images/check');
    S.imgResults = r.results || [];
    renderCheck();
  } catch (e) { toast(e.message, 'err'); }
}

/* ------------------------------------------------------------------ 5. Generálás */

function renderGenerate() {
  const p = pane('generalas');
  const o = S.state.output;
  const dir = h('input', { class: 'inp', value: o.dir || '', spellcheck: false, placeholder: 'pl. C:\\Users\\…\\Dokumentumok\\Energofish hírlevelek' });
  dir.addEventListener('input', () => { o.dir = dir.value; syncSoon(); });
  const pattern = h('input', { class: 'inp', value: o.filePattern || '', spellcheck: false, placeholder: '{sorszam}_{email}' });
  const example = h('div', { class: 'example' });
  const updEx = () => {
    const ps = S.excel && S.excel.partners.length ? S.excel.partners[0] : S.sample;
    const map = { sorszam: '001', email: ps.email, nev: ps.name, ceg: ps.company, kepviselo: ps.repName };
    const v = (pattern.value || '{sorszam}_{email}').replace(/\{([^{}]+)\}/g, (m, k) => { const n = norm(k).replace(/[^a-z0-9]/g, ''); return n in map ? (map[n] || '') : m; });
    example.replaceChildren('Példa: ', h('b', { text: 'html\\' + v.replace(/[<>:"/\\|?*]+/g, '_') + '.html' }));
  };
  pattern.addEventListener('input', () => { o.filePattern = pattern.value; updEx(); syncSoon(); });
  updEx();
  const from = h('input', { class: 'inp', value: o.from || '', placeholder: 'Energofish <hirlevel@energofish.hu>', spellcheck: false });
  from.addEventListener('input', () => { o.from = from.value; syncSoon(); });
  const fromRow = h('div', { class: 'field', style: { marginTop: '10px' } }, h('div', { class: 'field-top' }, h('label', { text: 'Feladó (opcionális)' })), from,
    h('div', { class: 'help', text: 'Üresen hagyva Outlook a saját fiókodat teszi be feladónak.' }));
  fromRow.classList.toggle('hidden', !o.eml);
  const eml = h('input', { type: 'checkbox', checked: !!o.eml, onchange: e => { o.eml = e.target.checked; fromRow.classList.toggle('hidden', !o.eml); syncSoon(0); } });

  p.replaceChildren(
    secHead('05 / Generálás', 'Hírlevelek elkészítése',
      'Partnerenként egy kész HTML fájl készül a kiválasztott sablonnal – ez mehet a küldőrendszerbe. Kérésre Outlookban megnyitható, azonnal küldhető EML piszkozat is készül.'),
    templatesCard(),
    h('div', { class: 'card card-pad' },
      h('div', { class: 'card-title' }, icon('folder'), 'Kimenet'),
      h('p', { class: 'card-sub', text: 'Minden generálás egy új, dátummal jelölt almappába kerül: html\\ (a levelek), eml\\ (ha kéred), áttekintő oldal, küldési lista (CSV) és a felhasznált tartalom.' }),
      h('div', { class: 'field' }, h('div', { class: 'field-top' }, h('label', { text: 'Kimeneti mappa' })),
        h('div', { class: 'path-row' }, dir,
          h('button', { class: 'btn btn-outline btn-sm', onclick: e => busy(e.currentTarget, async () => {
            try { const r = await api('/api/output/browse'); if (!r.cancelled) { o.dir = r.dir; dir.value = r.dir; syncSoon(0); } } catch (err) { toast(err.message, 'err'); }
          }) }, icon('folder', 16), 'Tallózás…'),
          h('button', { class: 'btn btn-ghost btn-sm', title: 'Mappa megnyitása', onclick: () => openPath(o.dir, true) }, icon('external', 16)))),
      h('div', { class: 'field', style: { marginTop: '14px' } }, h('div', { class: 'field-top' }, h('label', { text: 'Fájlnév minta' })), pattern,
        h('div', { class: 'help' }, 'Változók: ', ['{sorszam}', '{email}', '{nev}', '{ceg}', '{kepviselo}'].map((t, i) => [i ? ' ' : '', h('span', { class: 'tok', text: t })])), example)),
    h('div', { class: 'card card-pad' },
      h('div', { class: 'card-title' }, icon('box'), 'Formátumok'),
      h('label', { class: 'check-row' }, h('input', { type: 'checkbox', checked: true, disabled: true }),
        h('div', null, h('div', { class: 't', text: 'HTML fájlok (mindig)' }), h('div', { class: 'd', text: 'A küldőrendszerbe (pl. Mailchimp, SendGrid, saját rendszer) betölthető, kész levél.' }))),
      h('label', { class: 'check-row' }, eml,
        h('div', null, h('div', { class: 't', text: 'EML piszkozatok is' }), h('div', { class: 'd', text: 'Dupla kattintással Outlookban nyílik meg, címzettel és tárggyal kitöltve, azonnal elküldhető.' }))),
      fromRow),
    h('div', { id: 'genSummary' }),
    h('div', { id: 'genResult' }));
  renderGenSummary();
  renderGenResult();
}

function genCount() {
  if (!S.excel) return 0;
  let n = 0;
  for (const i of S.sel) if (!isBlocked(i)) n++;
  return n;
}

function renderGenSummary() {
  const box = $('#genSummary');
  if (!box) return;
  const n = genCount();
  const why = [];
  const cErr = S.issues.content.filter(i => i.level === 'error').length;
  if (!S.excel) why.push('Előbb töltsd be a partnerlistát (1. lépés).');
  else if (!S.sel.size) why.push('Nincs kijelölt partner.');
  if (cErr) why.push(`${cErr} hiba van a tartalomban vagy a termékeknél – lásd az Ellenőrzés lépést.`);
  if (!String(S.state.output.dir || '').trim()) why.push('Add meg a kimeneti mappát.');
  const skipped = S.excel ? Array.from(S.sel).filter(isBlocked).length : 0;
  const t = S.templates.find(t => t.id === S.state.template);
  const btn = h('button', { class: 'btn btn-primary btn-lg', disabled: !!why.length || !n || S.genBusy, onclick: doGenerate },
    S.genBusy ? h('span', { class: 'spin' }) : icon('zap'), S.genBusy ? 'Generálás…' : 'Generálás');
  box.replaceChildren(
    h('div', { class: 'gen-box' },
      h('div', null, h('div', { class: 'num', text: n }), h('div', { class: 'lbl', text: 'hírlevél' })),
      h('div', { class: 'grow' },
        h('div', { class: 'ttl', text: `${t ? t.name + ' (' + t.short + ')' : ''} · ${S.state.products.filter(p => p.on).length} termék` }),
        h('div', { class: 'why', text: why.length ? why.join(' ') : (skipped ? `${skipped} kijelölt partner hibás e-mail cím miatt kimarad.` : 'Minden kijelölt partner megkapja a saját változatát.') })),
      btn),
    !String(S.state.content['assets.base'] || '').trim()
      ? h('div', { class: 'note warn' }, icon('alert'), h('div', null, h('b', { text: 'A képtár webcíme üres. ' }), 'A levelek helyben jól látszanak (a képek a kimeneti mappába másolódnak), de kiküldés előtt az assets mappát fel kell tölteni egy https tárhelyre, és a címét megadni az Alapadatok között.'))
      : null);
}

async function doGenerate() {
  S.genBusy = true;
  renderGenSummary();
  try {
    await syncNow();
    const only = Array.from(S.sel).sort((a, b) => a - b);
    let r = await api('/api/generate', { only });
    if (r.syncFailed) {
      // B2B partnertörzs: a küldés előtti frissítés nem sikerült
      const when = S.excel.b2b ? fmtTime(S.excel.b2b.syncedAt) : '';
      if (!await confirmBox('A partnertörzs nem frissíthető', `${r.syncFailed}\n\nGenerálod a legutóbb (${when}) letöltött adatokkal? Aki azóta leiratkozott, még szerepelhet – ilyenkor küldés előtt mindenképp frissíts.`, 'Generálás a régi adatokkal', true)) return;
      r = await api('/api/generate', { only, skipSync: true });
    } else if (r.sync) {
      const sy = r.sync;
      applyExcel(r, true);
      S.sel = new Set(sy.only || []);
      renderData();
      const parts = [];
      if (sy.dropped) parts.push(`${sy.dropped} leiratkozott partner kimaradt`);
      if (sy.added) parts.push(`${sy.added} új partner bekerült`);
      toast('Partnertörzs frissítve a generálás előtt' + (parts.length ? ': ' + parts.join(', ') : ' – nem változott a címzettlista.'), parts.length ? 'warn' : 'ok', { timeout: 9000 });
      r = r.result;
    }
    S.lastResult = r;
    toast(`${r.generated} hírlevél elkészült.`, 'ok', { actions: [{ label: 'Mappa megnyitása', fn: () => openPath(r.folder) }, { label: 'Áttekintő', fn: () => openPath(r.index) }] });
  } catch (e) {
    toast(e.message, 'err');
  } finally {
    S.genBusy = false;
    renderGenSummary();
    renderGenResult();
  }
}

function renderGenResult() {
  const box = $('#genResult');
  if (!box) return;
  const r = S.lastResult;
  if (!r) { box.replaceChildren(); return; }
  const secs = (r.millis / 1000).toFixed(1).replace('.', ',');
  box.replaceChildren(h('div', { class: 'card card-pad result' },
    h('div', { class: 'result-head' }, h('div', { class: 'big-ok' }, icon('check', 26)),
      h('div', null, h('h3', { text: `${r.generated} hírlevél elkészült` }), h('div', { class: 'p-sub', text: `${secs} mp alatt` + (r.skipped && r.skipped.length ? ` · ${r.skipped.length} partner kimaradt` : '') }))),
    h('div', { class: 'path', text: r.folder }),
    h('div', { class: 'btn-row', style: { marginTop: '14px' } },
      h('button', { class: 'btn btn-primary btn-sm', onclick: () => openPath(r.folder) }, icon('folder', 16), 'Mappa megnyitása'),
      h('button', { class: 'btn btn-outline btn-sm', onclick: () => openPath(r.index) }, icon('table', 16), 'Áttekintő'),
      r.firstHtml ? h('button', { class: 'btn btn-outline btn-sm', onclick: () => openPath(r.firstHtml) }, icon('eye', 16), 'Első levél') : null,
      h('button', { class: 'btn btn-ghost btn-sm', onclick: () => openPath(r.csv) }, icon('sheet', 16), 'Küldési lista (CSV)')),
    r.warnings && r.warnings.length ? h('ul', null, r.warnings.map(w => h('li', { text: w }))) : null,
    r.skipped && r.skipped.length ? [h('div', { class: 'label-caps', style: { marginTop: '14px' }, text: 'Kimaradt partnerek' }),
      h('ul', null, r.skipped.map(s => h('li', { text: `${s.row}. sor: ${s.name || s.email || '–'} – ${s.reason}` })))] : null));
}

/* ------------------------------------------------------------------ sablonok */

function templatesCard() {
  const cards = S.templates.map(t => {
    const actions = t.custom ? h('div', { class: 'tpl-actions' },
      h('button', { class: 'btn btn-ghost btn-sm', title: 'Átnevezés', onclick: e => { e.stopPropagation(); renameTemplate(t); } }, icon('pen', 15)),
      h('button', { class: 'btn btn-ghost btn-sm btn-danger', title: t.overrides ? 'Törlés (az eredeti beépített tér vissza)' : 'Törlés', onclick: e => { e.stopPropagation(); deleteTemplate(t); } }, icon('trash', 15))) : null;
    return h('div', { class: 'tpl-card' + (t.id === S.state.template ? ' sel' : ''), role: 'button', tabindex: '0', onclick: () => setTemplate(t.id),
      onkeydown: e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setTemplate(t.id); } } },
      tplThumb(t), actions,
      h('div', { class: 'nm' }, h('span', { class: 'pill dark', text: t.short || '–' }), h('span', { class: 'tn', text: t.name, title: t.name })),
      h('div', { class: 'badges' },
        t.overrides ? h('span', { class: 'pill ok', text: 'frissített' }) : t.custom ? h('span', { class: 'pill ok', text: 'hozzáadott' }) : h('span', { class: 'pill muted', text: 'beépített' }),
        t.hasPoll ? h('span', { class: 'pill muted', text: 'kérdés-blokk' }) : null,
        t.fixedSlots ? h('span', { class: 'pill warn', text: `fix ${t.fixedSlots} termék` }) : null),
      h('div', { class: 'ds', text: t.desc || '' }),
      t.report && t.report.length ? h('div', { class: 'tpl-notes' }, t.report.map(n => h('div', null, icon('alert', 13), h('span', { text: n })))) : null);
  });
  const add = h('div', { class: 'tpl-card add', role: 'button', tabindex: '0', title: 'Sablon hozzáadása', onclick: () => importTemplates(),
    onkeydown: e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); importTemplates(); } } },
    h('div', { class: 'add-ic' }, icon('plus', 26)),
    h('div', { class: 'nm' }, 'Sablon hozzáadása'),
    h('div', { class: 'ds', text: '.html sablon vagy a tervezőtől kapott .zip csomag – az ablakba húzva is működik.' }));
  return h('div', { class: 'card card-pad' },
    h('div', { class: 'card-title' }, icon('mail'), 'Sablon'),
    h('p', { class: 'card-sub', text: 'Válaszd ki, melyik sablonnal készüljenek a levelek. Új sablon a tervezői formátumban ({{kulcs}} helyőrzőkkel) adható hozzá, és a gépen megmarad. Ha a v1/v2/v4 új változatát kapod, ugyanazzal a fájlnévvel hozzáadva frissíti a beépítettet; törlésével az eredeti tér vissza.' }),
    h('div', { class: 'tpl-cards' }, cards, add));
}

function tplThumb(t) {
  const box = h('div', { class: 'tpl-thumb' });
  const fr = h('iframe', { sandbox: 'allow-same-origin', tabindex: '-1', title: t.name });
  box.append(fr);
  const fit = () => { if (box.clientWidth) fr.style.transform = `scale(${box.clientWidth / 660})`; };
  new ResizeObserver(fit).observe(box);
  fetch(`/api/preview?p=${S.pv}&tpl=${encodeURIComponent(t.id)}`, { headers: { 'X-Token': TOKEN } })
    .then(r => r.text()).then(html => { fr.srcdoc = html; }).catch(() => {});
  return box;
}

function applyTemplates(r, selectId) {
  S.templates = r.templates;
  if (r.issues) setIssues(r.issues);
  if (selectId && S.templates.some(t => t.id === selectId)) S.state.template = selectId;
  else if (r.template) S.state.template = r.template;
  renderTemplateSeg();
  updateTemplateDependent();
  if (S.tab === 'generalas') renderGenerate();
  syncNow();
}

function showImportResults(results) {
  const ok = results.filter(x => x.ok), bad = results.filter(x => !x.ok);
  const body = h('div', { class: 'imp-list' }, results.map(x => {
    const notes = x.notes || [];
    const st = !x.ok ? 'err' : notes.length ? 'warn' : 'ok';
    const sub = !x.ok ? x.error : x.override ? `${x.file} · a beépített sablon frissítve` : x.replaced ? `${x.file} · a meglévő sablon frissítve` : `${x.file} · új sablon`;
    return h('div', { class: 'imp' },
      h('span', { class: 'st ' + st }, icon(st === 'err' ? 'error' : st === 'warn' ? 'alert' : 'check', 15)),
      h('div', null,
        h('div', { class: 'p-name', text: x.ok ? x.name : x.file }),
        h('div', { class: 'p-sub', text: sub }),
        notes.map(n => h('div', { class: 'p-sub warn-t', text: n }))));
  }));
  infoModal(bad.length && !ok.length ? 'A sablon nem adható hozzá' : `${ok.length} sablon hozzáadva`, body);
}

function afterImport(r) {
  const first = (r.results || []).find(x => x.ok && !x.override);
  applyTemplates(r, first ? first.id : null);
  if (S.tab !== 'generalas') setTab('generalas');
  showImportResults(r.results || []);
}

async function importTemplates() {
  try {
    const r = await api('/api/templates/import');
    if (!r.cancelled) afterImport(r);
  } catch (e) { toast(e.message, 'err'); }
}

async function uploadTemplates(files) {
  let last = null;
  const results = [];
  for (const f of files) {
    try {
      last = await api('/api/templates/upload', f, { raw: true, headers: { 'X-Filename': encodeURIComponent(f.name) } });
      results.push(...(last.results || []));
    } catch (e) { results.push({ file: f.name, ok: false, error: e.message }); }
  }
  if (last) afterImport(Object.assign({}, last, { results }));
  else showImportResults(results);
}

async function renameTemplate(t) {
  const v = await formModal('Sablon átnevezése', [
    { k: 'name', label: 'Név', value: t.name },
    { k: 'desc', label: 'Leírás', value: t.desc || '' }], 'Mentés');
  if (!v) return;
  try { applyTemplates(await api('/api/templates/update', { id: t.id, name: v.name, desc: v.desc })); } catch (e) { toast(e.message, 'err'); }
}

async function deleteTemplate(t) {
  const txt = t.overrides
    ? `A(z) „${t.name}” frissített változata törlődik, és a programba épített eredeti sablon tér vissza.`
    : `A(z) „${t.name}” sablon törlődik a gépről. A már legenerált levelek nem változnak.`;
  if (!await confirmBox('Sablon törlése', txt, 'Törlés', true)) return;
  try { applyTemplates(await api('/api/templates/delete', { id: t.id })); toast('A sablon törölve.', 'ok'); } catch (e) { toast(e.message, 'err'); }
}

function infoModal(title, content) {
  const done = () => { bg.remove(); document.removeEventListener('keydown', key); };
  const key = e => { if (e.key === 'Escape') done(); };
  const bg = h('div', { class: 'modal-bg', onclick: e => { if (e.target === bg) done(); } },
    h('div', { class: 'modal wide', role: 'dialog' }, h('div', { class: 'mh', text: title }), h('div', { class: 'mb' }, content),
      h('div', { class: 'mf' }, h('button', { class: 'btn btn-primary', text: 'Rendben', onclick: done }))));
  document.addEventListener('keydown', key);
  document.body.append(bg);
  $('.mf .btn', bg).focus();
}

function formModal(title, fields, okLabel) {
  return new Promise(resolve => {
    const inputs = {};
    const done = v => { bg.remove(); resolve(v); };
    const submit = () => { const v = {}; for (const f of fields) v[f.k] = inputs[f.k].value; done(v); };
    const bg = h('div', { class: 'modal-bg' },
      h('div', { class: 'modal', role: 'dialog' }, h('div', { class: 'mh', text: title }),
        h('div', { class: 'mb' }, fields.map(f => h('div', { class: 'field', style: { marginBottom: '12px' } },
          h('div', { class: 'field-top' }, h('label', { text: f.label })),
          inputs[f.k] = h('input', { class: 'inp', value: f.value, onkeydown: e => { if (e.key === 'Enter') submit(); if (e.key === 'Escape') done(null); } })))),
        h('div', { class: 'mf' }, h('button', { class: 'btn btn-ghost', text: 'Mégse', onclick: () => done(null) }), h('button', { class: 'btn btn-primary', text: okLabel, onclick: submit }))));
    document.body.append(bg);
    inputs[fields[0].k].focus();
    inputs[fields[0].k].select();
  });
}

/* ------------------------------------------------------------------ előnézet */

function buildPreview() {
  const a = $('#preview');
  const devBtn = (id, ic, title) => h('button', { dataset: { dev: id }, title, onclick: () => setDevice(id) }, icon(ic, 17));
  a.append(
    h('div', { class: 'pv-top' },
      h('div', { class: 'seg', id: 'tplSeg' }),
      h('div', { class: 'grow' }),
      h('div', { class: 'seg icons', id: 'devSeg' }, devBtn('desktop', 'monitor', 'Asztali nézet'), devBtn('tablet', 'tablet', 'Tablet nézet (768 px)'), devBtn('mobile', 'phone', 'Mobil nézet (390 px, 2 oszlopos rács)'))),
    h('div', { class: 'pv-bar' },
      h('button', { class: 'sq', title: 'Előző partner', onclick: () => stepPartner(-1) }, icon('left')),
      h('select', { id: 'pvSel', onchange: e => setPreviewPartner(+e.target.value) }),
      h('button', { class: 'sq', title: 'Következő partner', onclick: () => stepPartner(1) }, icon('right')),
      h('button', { class: 'sq', title: 'Előnézet frissítése', onclick: () => refreshPreview() }, icon('refresh', 16)),
      h('button', { class: 'sq', title: 'Megnyitás a böngészőben', onclick: openPreviewInBrowser }, icon('external', 16))),
    h('div', { class: 'inbox', title: 'Így jelenik meg a levél a postafiókban' },
      h('div', { class: 'av' }, h('img', { src: '/static/img/energofish-mark-light.png', alt: '' })),
      h('div', { class: 'meta' },
        h('div', { class: 'from' }, h('b', { id: 'pvFrom', text: 'Energofish' }), h('span', { text: 'most' })),
        h('div', { class: 'subj', id: 'pvSubj' }),
        h('div', { class: 'pre', id: 'pvPre' }))),
    h('div', { class: 'pv-canvas', id: 'pvCanvas' },
      h('div', { class: 'pv-loading', id: 'pvLoad' }, h('span', { class: 'spin' })),
      h('div', { id: 'pvSizer', style: { position: 'relative', margin: '0 auto' } },
        h('div', { class: 'pv-stage', id: 'pvStage', style: { position: 'absolute', left: 0, top: 0 } },
          h('iframe', { id: 'pvFrame', title: 'Hírlevél előnézet', sandbox: 'allow-same-origin' })))),
    h('div', { class: 'linkbar', id: 'linkbar' }, icon('link', 14), h('span', { class: 'u', id: 'linkUrl' })));
  S.device = ['desktop', 'tablet', 'mobile'].includes(ls('device')) ? ls('device') : 'desktop';
  setDevice(S.device, true);
  new ResizeObserver(() => fitFrame()).observe($('#pvCanvas'));
}

function renderTemplateSeg() {
  const seg = $('#tplSeg');
  seg.replaceChildren(...S.templates.map(t => h('button', { class: t.id === S.state.template ? 'on' : '', title: t.desc + (t.custom ? ' (hozzáadott sablon)' : ''), onclick: () => setTemplate(t.id) },
    h('span', { class: 'sh', text: (t.short || '').toUpperCase() }), t.name, t.custom ? h('span', { class: 'dot', title: t.overrides ? 'frissített változat' : 'hozzáadott sablon' }) : null)));
}

function setTemplate(id) {
  if (S.state.template === id) return;
  S.state.template = id;
  renderTemplateSeg();
  updateTemplateDependent();
  if (S.tab === 'generalas') renderGenerate();
  syncNow();
}

function setDevice(id, silent) {
  S.device = id;
  ls('device', id);
  $$('#devSeg button').forEach(b => b.classList.toggle('on', b.dataset.dev === id));
  $('#pvStage').classList.toggle('device', id !== 'desktop');
  if (!silent) { fitFrame(); refreshPreview(); }
}

function renderPreviewSelect() {
  const sel = $('#pvSel');
  if (!S.excel || !S.excel.partners.length) {
    sel.replaceChildren(h('option', { value: -1, text: `Minta partner – ${S.sample.name} (${S.sample.email})` }));
    sel.value = -1;
    return;
  }
  const n = S.excel.partners.length;
  const idx = new Set(Array.from({ length: Math.min(n, MAX_PREVIEW_OPTIONS) }, (_, i) => i));
  if (S.pv >= 0) idx.add(S.pv);
  sel.replaceChildren(...Array.from(idx).sort((a, b) => a - b).map(i => { const p = S.excel.partners[i]; return h('option', { value: i, text: `${i + 1}. ${p.name || p.company || '(név nélkül)'} – ${p.email || 'nincs e-mail'}${isBlocked(i) ? '  ⚠' : ''}` }); }),
    n > MAX_PREVIEW_OPTIONS ? h('option', { disabled: true, text: `… további ${n - MAX_PREVIEW_OPTIONS} partner (a nyilakkal léptethető)` }) : null);
  sel.value = S.pv;
}

function setPreviewPartner(i) {
  if (!S.excel || i < 0 || i >= S.excel.partners.length) return;
  S.pv = i;
  $('#pvSel').value = i;
  $$('#ptBody tr.row').forEach(tr => tr.classList.toggle('current', +tr.dataset.i === i));
  $$('.pane[data-pane="tartalom"] .inp').forEach(el => el._recount && el._recount());
  refreshPreview();
}

function stepPartner(d) {
  if (!S.excel || !S.excel.partners.length) return;
  const n = S.excel.partners.length;
  setPreviewPartner(((S.pv < 0 ? 0 : S.pv) + d + n) % n);
}

function frameWidth() { return { mobile: 390, tablet: 768 }[S.device] || 660; }

function fitFrame() {
  const frame = $('#pvFrame'), canvas = $('#pvCanvas'), stage = $('#pvStage'), sizer = $('#pvSizer');
  if (!frame || !canvas) return;
  const W = frameWidth();
  let H = 900;
  try {
    const d = frame.contentDocument;
    if (d && d.documentElement) H = Math.max(d.documentElement.scrollHeight, d.body ? d.body.scrollHeight : 0, 300);
  } catch (e) { /* nincs hozzáférés */ }
  const avail = canvas.clientWidth - 32 - (S.device !== 'desktop' ? 24 : 0);
  const s = Math.min(1, Math.max(0.3, avail / W));
  frame.style.width = W + 'px';
  frame.style.height = H + 'px';
  stage.style.width = W + 'px';
  stage.style.transform = `scale(${s})`;
  stage.style.transformOrigin = 'top left';
  const extra = S.device !== 'desktop' ? 24 : 0;
  sizer.style.width = (W * s + extra) + 'px';
  sizer.style.height = (H * s + 20 + extra) + 'px';
  stage.style.left = (extra / 2) + 'px';
}

async function refreshPreview() {
  const seq = ++S.pvSeq;
  const load = $('#pvLoad');
  load.classList.add('on');
  try {
    const r = await fetch(`/api/preview?p=${S.pv}&tpl=${encodeURIComponent(S.state.template)}`, { headers: { 'X-Token': TOKEN } });
    const html = await r.text();
    if (seq !== S.pvSeq) return;
    if (!r.ok) throw new Error(html);
    const doc = new DOMParser().parseFromString(html, 'text/html');
    $('#pvSubj').textContent = doc.title || '(nincs tárgy)';
    const pre = doc.body.querySelector('div');
    $('#pvPre').textContent = pre ? pre.textContent.replace(PREHEADER_FILLER, '').trim() : '';
    const frame = $('#pvFrame');
    const canvas = $('#pvCanvas');
    const top = canvas.scrollTop;
    frame.onload = () => {
      if (seq !== S.pvSeq) return;
      fitFrame();
      canvas.scrollTop = top;
      wireFrame(frame);
      load.classList.remove('on');
      setTimeout(fitFrame, 400);
    };
    frame.srcdoc = html;
  } catch (e) {
    if (seq === S.pvSeq) {
      load.classList.remove('on');
      toast('Előnézeti hiba: ' + e.message, 'err');
    }
  }
}

function wireFrame(frame) {
  let d;
  try { d = frame.contentDocument; } catch (e) { return; }
  if (!d) return;
  const bar = $('#linkbar'), url = $('#linkUrl');
  d.addEventListener('mouseover', e => {
    const a = e.target.closest && e.target.closest('a');
    if (a) { url.textContent = a.getAttribute('href') || ''; bar.classList.add('on'); }
  });
  d.addEventListener('mouseout', e => { if (e.target.closest && e.target.closest('a')) bar.classList.remove('on'); });
  d.addEventListener('click', e => {
    const a = e.target.closest && e.target.closest('a');
    if (!a) return;
    e.preventDefault();
    const href = a.getAttribute('href') || '';
    if (href.startsWith('#leiratkozo')) {
      toast('A partner saját leiratkozó linkje – az előnézetben letiltva, mert a megnyitása azonnal leiratkoztatná. A kész levélben a valódi link szerepel.', 'info', { timeout: 8000 });
      return;
    }
    const acts = [];
    if (/^(https?:|mailto:|tel:)/i.test(href)) acts.push({ label: 'Megnyitás', fn: () => api('/api/openurl', { url: href }).catch(err => toast(err.message, 'err')) });
    acts.push({ label: 'Másolás', fn: () => navigator.clipboard && navigator.clipboard.writeText(href) });
    toast('A levélben ez a link szerepel:', 'info', { url: href, actions: acts, timeout: 8000 });
  });
  d.querySelectorAll('img').forEach(img => img.addEventListener('load', () => fitFrame()));
}

async function openPreviewInBrowser() {
  await syncNow();
  const u = `${location.origin}/elonezet?p=${S.pv}&tpl=${encodeURIComponent(S.state.template)}&t=${TOKEN}`;
  try { await api('/api/openurl', { url: u }); } catch (e) { toast(e.message, 'err'); }
}

/* ------------------------------------------------------------------ helyi menü a mezőkhöz */

function setupContextMenu() {
  let menu = null;
  const close = () => { if (menu) { menu.remove(); menu = null; } };
  document.addEventListener('contextmenu', e => {
    const el = e.target.closest('input[type="text"], input[type="search"], input:not([type]), textarea');
    if (S.mode !== 'webview') return;
    e.preventDefault();
    close();
    if (!el || el.disabled) return;
    el.focus();
    const hasSel = el.selectionStart !== el.selectionEnd;
    const item = (label, fn, enabled = true) => h('button', { disabled: !enabled, onclick: async () => { close(); await fn(); el.focus(); } }, label);
    menu = h('div', { class: 'menu ctx' },
      item('Kivágás', () => { document.execCommand('cut'); }, hasSel),
      item('Másolás', () => { document.execCommand('copy'); }, hasSel),
      item('Beillesztés', async () => {
        try {
          const t = await navigator.clipboard.readText();
          el.setRangeText(t, el.selectionStart, el.selectionEnd, 'end');
          el.dispatchEvent(new Event('input'));
        } catch (err) { toast('A vágólap nem olvasható: használd a Ctrl+V-t.', 'warn'); }
      }),
      h('hr'),
      item('Mindent kijelöl', () => el.select()));
    document.body.append(menu);
    const r = menu.getBoundingClientRect();
    menu.style.left = Math.min(e.clientX, innerWidth - r.width - 8) + 'px';
    menu.style.top = Math.min(e.clientY, innerHeight - r.height - 8) + 'px';
  });
  document.addEventListener('mousedown', e => { if (menu && !menu.contains(e.target)) close(); });
  document.addEventListener('keydown', e => { if (e.key === 'Escape') close(); });
  window.addEventListener('blur', close);
}

/* ------------------------------------------------------------------ indulás */

async function quitApp() {
  try { await api('/api/quit'); } catch (e) { /* már leállt */ }
  document.body.replaceChildren(h('div', { style: { display: 'grid', placeItems: 'center', height: '100vh', background: '#1A171E', color: '#fff', textAlign: 'center' } },
    h('div', null, h('img', { src: '/static/img/energofish-mark-light.png', alt: '', width: 63, height: 52 }), h('h2', { text: 'A program leállt.' }), h('p', { style: { color: '#ccc' }, text: 'Ez a böngészőlap bezárható.' }))));
}

/* ------------------------------------------------------------------ B2B partnertörzs (1.3) */

const B2B = { state: null, busy: false };
const B2B_TRI = [['', 'Mind'], ['only', 'Csak ők'], ['exclude', 'Nélkülük']];
const B2B_GREET = [['name', 'A partner nevével', '„Kedves JDB Hungary Zrt.!”, „Kedves Kiss Péter!” – a Tartalom › Megszólítás mezője szerint'],
  ['auto', 'Cégeknek tartalék', 'Cégnévnél (csupa nagybetű vagy Kft., Bt., Zrt. …) „Kedves Partnerünk!”, személynévnél a név'],
  ['fallback', 'Mindenkinek tartalék', 'Mindenki a Tartalom › tartalék megszólítást kapja']];

async function b2bLoadState() {
  B2B.state = await api('/api/b2b/state');
  return B2B.state;
}

function b2bGroup(id) {
  const st = B2B.state;
  return st && st.groups.find(g => g.id === id);
}

function b2bSyncText(g) {
  if (!g) return '';
  if (!g.syncedAt) return g.configured ? 'még nem volt letöltve' : 'nincs megadva forrás';
  return `${g.mailable} levelezhető partner · szinkron: ${fmtTime(g.syncedAt)}`;
}

function b2bLogText(l) {
  if (!l) return '';
  if (!l.ok) return 'Az utolsó frissítés megszakadt: ' + l.result.replace(/^megszakítva: /, '');
  const parts = [`${l.records} rekord`];
  if (l.new) parts.push(`+${l.new} új`);
  if (l.changed) parts.push(`${l.changed} adata változott`);
  if (l.reactivated) parts.push(`${l.reactivated} újra feliratkozott`);
  if (l.inactivated) parts.push(`${l.inactivated} leiratkozott / törölt → inaktív`);
  return parts.join(' · ');
}

// Az Adatok lépés kártyája (ha még nincs betöltött partnerlista).
function b2bCard() {
  const card = h('div', { class: 'card card-pad b2b-card' });
  const body = h('div', { class: 'b2b-card-body' }, h('div', { class: 'p-sub', text: 'Partnertörzs betöltése…' }));
  card.append(
    h('div', { class: 'b2b-card-head' },
      h('div', { class: 'file-ic' }, icon('users', 26)),
      h('div', { style: { flex: 1, minWidth: 0 } },
        h('div', { class: 'card-title', style: { margin: 0 } }, 'B2B partnertörzs'),
        h('div', { class: 'p-sub', text: 'A webshop feliratkozói célcsoportonként (ország). Képviselő, megye, besorolás és más tulajdonságok alapján állíthatsz össze partnerhalmazt.' }))),
    body);
  b2bLoadState().then(st => {
    const g = b2bGroup(st.settings.group) || st.groups[0];
    const configured = st.groups.filter(x => x.configured);
    body.replaceChildren(
      h('div', { class: 'b2b-groups' }, st.groups.filter(x => x.configured || x.syncedAt).map(x =>
        h('span', { class: 'pill ' + (x.syncedAt ? 'ok' : 'muted'), title: b2bSyncText(x), text: `${x.label}${x.syncedAt ? ' · ' + x.mailable : ''}` }))),
      !configured.length ? h('div', { class: 'note cream' }, icon('info'), h('div', null, 'Még nincs megadva forrás. A ', h('b', { text: 'Források' }), ' gombnál illeszd be a célcsoportok linkjeit (pl. „B2B HU: https://…&token=…”). A tokeneket a program titkosítva tárolja ezen a gépen.')) : null,
      h('div', { class: 'btn-row' },
        h('button', { class: 'btn btn-primary', onclick: () => openPartnerSet(g && g.id) }, icon('users'), 'Partnerhalmaz összeállítása…'),
        h('button', { class: 'btn btn-ghost', onclick: () => openSources() }, icon('link', 16), 'Források…')),
      h('div', { class: 'p-sub', text: 'A böngészőben letöltött exportot (JSON) is behúzhatod az ablakba; a forráslinkeket pedig egy „B2B HU: https://…” sorokat tartalmazó .txt fájl behúzásával is felveheted.' }));
  }).catch(e => body.replaceChildren(h('div', { class: 'note err' }, icon('error'), h('div', { text: e.message }))));
  return card;
}

// A betöltött halmaz kártyája (a fájlkártya helyett).
function b2bFileCard() {
  const ex = S.excel, info = ex.b2b || {};
  return h('div', { class: 'card card-pad' },
    h('div', { class: 'file-card' },
      h('div', { class: 'file-ic' }, icon('users', 26)),
      h('div', { style: { minWidth: 0, flex: 1 } },
        h('div', { class: 'file-name', text: ex.fileName }),
        h('div', { class: 'file-sum', text: info.summary || '' }),
        h('div', { class: 'file-meta', text: `${info.selected} partner a halmazban (${info.active} aktívból) · partnertörzs szinkron: ${fmtTime(info.syncedAt)}` })),
      h('button', { class: 'btn btn-ghost btn-sm', title: 'Lista bezárása', onclick: closeExcel, style: { alignSelf: 'flex-start' } }, icon('x', 16))),
    h('div', { class: 'note info', style: { margin: '12px 0 0' } }, icon('shield'),
      h('div', { text: 'Generáláskor a program előbb automatikusan frissíti a partnertörzset: aki közben leiratkozott, kimarad, az új feliratkozók (ha illenek a feltételekre) bekerülnek. A leiratkozó linkeket soha nem nyitja meg.' })),
    h('div', { class: 'file-actions' },
      h('button', { class: 'btn btn-primary btn-sm', onclick: () => openPartnerSet(info.group, info) }, icon('pen', 16), 'Halmaz módosítása…'),
      h('button', { class: 'btn btn-outline btn-sm', dataset: { busy: 'Frissítés…' }, onclick: e => busy(e.currentTarget, () => b2bRefreshLoaded(info)) }, icon('refresh', 16), 'Frissítés most'),
      h('button', { class: 'btn btn-ghost btn-sm', onclick: () => openRepPhotos(info.group) }, icon('user', 16), 'Képviselő-fotók…'),
      h('button', { class: 'btn btn-ghost btn-sm', onclick: e => busy(e.currentTarget, browseExcel) }, icon('open', 16), 'Excel helyette…')));
}

async function b2bRefreshLoaded(info) {
  const r = await api('/api/b2b/sync', { group: info.group });
  if (r.failed) { b2bSyncFailed(r, info.group, () => b2bRefreshLoaded(info)); return; }
  toast('Partnertörzs frissítve: ' + b2bLogText(r.result.log), 'ok');
  applyExcel(await api('/api/b2b/load', { group: info.group, filter: info.filter, name: info.name }), true);
}

async function b2bSyncFailed(r, group, retry) {
  if (r.suspicious) {
    if (await confirmBox('Gyanúsan kevés partner', r.failed + '\n\nHa biztos vagy benne, hogy ennyien maradtak (pl. tömeges leiratkozás vagy tisztítás), a frissítés kényszeríthető: a hiányzók inaktívak lesznek.', 'Mégis frissítem', true)) {
      const f = await api('/api/b2b/sync', { group, force: true });
      if (f.failed) toast(f.failed, 'err');
      else { toast('Partnertörzs frissítve: ' + b2bLogText(f.result.log), 'ok'); if (retry) retry(); }
    }
    return;
  }
  toast(r.failed, 'err', { timeout: 12000 });
}

// Böngészőben letöltött export (JSON) betöltése – ugyanazokkal a szabályokkal, mint a letöltés.
async function b2bUpload(file, group, force) {
  const res = await fetch('/api/b2b/import', { method: 'POST', headers: { 'X-Token': TOKEN, 'X-Group': group, 'X-Filename': encodeURIComponent(file.name), 'X-Force': force ? '1' : '0' }, body: file });
  let r = null;
  try { r = await res.json(); } catch (e) { /* üres */ }
  if (!res.ok || !r || r.error) throw new Error((r && r.error) || 'A fájl nem tölthető be (' + res.status + ')');
  B2B.state = r;
  if (r.failed) {
    if (r.suspicious && await confirmBox('Gyanúsan kevés partner', r.failed + '\n\nHa biztos vagy benne, hogy ez a teljes, friss export, a betöltés kényszeríthető: a hiányzók inaktívak lesznek.', 'Mégis betöltöm', true)) return b2bUpload(file, group, true);
    if (!r.suspicious) toast(r.failed, 'err', { timeout: 12000 });
    return null;
  }
  toast(`${file.name}: ` + b2bLogText(r.result.log), 'ok', { timeout: 9000 });
  return r;
}

async function b2bImportFile(file, group) {
  let st;
  try { st = await b2bLoadState(); } catch (e) { toast(e.message, 'err'); return null; }
  const ageH = (Date.now() - file.lastModified) / 36e5;
  if (!group) {
    const sel = h('select', { class: 'inp' }, st.groups.map(g => h('option', { value: g.id, selected: g.id === (st.settings.group || 'B2B_HU'), text: `${g.label} – ${g.country}` })));
    const ok = await new Promise(resolve => {
      const done = v => { bg.remove(); resolve(v); };
      const bg = h('div', { class: 'modal-bg' }, h('div', { class: 'modal', role: 'dialog' },
        h('div', { class: 'mh', text: 'Partnertörzs betöltése fájlból' }),
        h('div', { class: 'mb' },
          h('p', { style: { margin: '0 0 10px' }, text: `${file.name} (${(file.size / 1024).toFixed(0)} KB)` }),
          h('div', { class: 'label-caps', style: { marginBottom: '6px' }, text: 'Melyik célcsoport exportja?' }), sel,
          ageH > 24 ? h('div', { class: 'note warn', style: { margin: '12px 0 0' } }, icon('alert'), h('div', { text: `A fájl ${Math.round(ageH / 24)} napos. Régi exporttal az azóta leiratkozottak újra aktívvá válnának – lehetőleg friss exportot tölts be (generáláskor a program úgyis frissít a linkről, ha be van állítva).` })) : null,
          h('p', { class: 'p-sub', style: { margin: '12px 0 0' }, text: 'Ugyanazok a szabályok érvényesek, mint a letöltésnél: új partner bekerül, a meglévő frissül, aki nincs a fájlban, inaktív lesz.' })),
        h('div', { class: 'mf' }, h('button', { class: 'btn btn-ghost', text: 'Mégse', onclick: () => done(null) }), h('button', { class: 'btn btn-primary', text: 'Betöltés', onclick: () => done(sel.value) }))));
      document.body.append(bg);
    });
    if (!ok) return null;
    group = ok;
  }
  try {
    const r = await b2bUpload(file, group, false);
    if (r && pane('adatok') && !S.excel) renderData();
    return r;
  } catch (e) { toast(e.message, 'err'); return null; }
}

async function b2bImportSourcesText(file) {
  try {
    const text = await file.text();
    B2B.state = await api('/api/b2b/sources', { text });
    toast('Partnerforrások titkosítva elmentve: ' + (B2B.state.saved || []).join(', ') + '. A nyílt szöveges fájlt érdemes törölni.', 'ok', { timeout: 10000 });
    if (!S.excel) renderData();
  } catch (e) { toast(e.message, 'err'); }
}

// Partnerhalmaz-választó.
async function openPartnerSet(groupId, loaded) {
  let st;
  try { st = await b2bLoadState(); } catch (e) { toast(e.message, 'err'); return; }
  const set = st.settings;
  let group = groupId || set.group || 'B2B_HU';
  let filter = JSON.parse(JSON.stringify(loaded && loaded.filter ? loaded.filter : (set.filter || {})));
  let options = JSON.parse(JSON.stringify(set.options || {}));
  let presetName = (loaded && loaded.name) || '';
  let tab = '', last = null, seq = 0;
  const open = {};

  const groupSel = h('select', { class: 'inp ps-group' });
  const syncInfo = h('div', { class: 'ps-sync' });
  const left = h('div', { class: 'ps-left' });
  const right = h('div', { class: 'ps-right' });
  const loadBtn = h('button', { class: 'btn btn-primary', onclick: e => busy(e.currentTarget, doLoad) }, icon('check'), 'Betöltés');
  const done = () => { document.removeEventListener('keydown', key); bg.remove(); };
  const key = e => { if (e.key === 'Escape' && !document.querySelector('.modal-bg + .modal-bg')) done(); };

  function renderGroups() {
    groupSel.replaceChildren(...B2B.state.groups.map(g => h('option', { value: g.id, selected: g.id === group, text: `${g.label} – ${g.country}${g.configured ? '' : ' (nincs forrás)'}` })));
    const g = b2bGroup(group);
    syncInfo.replaceChildren(
      h('span', { class: 'dot ' + (g && g.syncedAt ? 'ok' : 'off') }),
      h('span', { text: b2bSyncText(g) }),
      g && g.lastLog ? h('span', { class: 'p-sub', title: (g.lastLog.warnings || []).join('\n'), text: ' · ' + b2bLogText(g.lastLog) }) : null);
  }

  const query = debounce(runQuery, 120);

  async function runQuery() {
    const my = ++seq;
    try {
      const r = await api('/api/b2b/query', { group, filter, options, list: tab, limit: 400 });
      if (my !== seq) return;
      last = r;
      renderLeft();
      renderRight();
    } catch (e) { toast(e.message, 'err'); }
  }

  function facet(title, key, notKey, values, opts = {}) {
    const sel = new Set(filter[key] || []);
    const isOpen = open[key] != null ? open[key] : (sel.size > 0 || opts.open);
    const head = h('button', { type: 'button', class: 'ps-fh', onclick: () => { open[key] = !isOpen; renderLeft(); } },
      icon('chev', 15), h('span', { text: title }),
      sel.size ? h('span', { class: 'pill dark', text: (filter[notKey] ? 'kivéve ' : '') + sel.size }) : null);
    const box = h('div', { class: 'ps-facet' + (isOpen ? ' open' : '') }, head);
    if (!isOpen) return box;
    box.append(h('div', { class: 'ps-fopts' },
      h('label', { class: 'ps-not', title: 'A kijelöltek kivételével mindenki' }, h('input', { type: 'checkbox', checked: !!filter[notKey], onchange: e => { filter[notKey] = e.target.checked || undefined; query(); } }), 'kivéve'),
      sel.size ? h('button', { type: 'button', class: 'linkbtn', text: 'törlés', onclick: () => { delete filter[key]; delete filter[notKey]; query(); } }) : null));
    box.append(h('div', { class: 'ps-vals' + (values.length > 8 ? ' many' : '') }, values.map(v => {
      const on = sel.has(v.value);
      return h('label', { class: 'ps-val' + (on ? ' on' : '') + (v.count ? '' : ' zero') },
        h('input', { type: 'checkbox', checked: on, onchange: e => {
          const s = new Set(filter[key] || []);
          e.target.checked ? s.add(v.value) : s.delete(v.value);
          filter[key] = Array.from(s);
          if (!filter[key].length) { delete filter[key]; delete filter[notKey]; }
          query();
        } }),
        h('span', { class: 'lbl', text: v.label, title: v.value && v.value !== v.label ? v.value + ' · ' + v.label : v.label }),
        h('span', { class: 'cnt', title: `${v.count} a többi feltétellel / ${v.total} összesen`, text: v.count === v.total ? v.total : `${v.count}/${v.total}` }));
    })));
    return box;
  }

  function tri(title, key, counts, help) {
    return h('div', { class: 'ps-tri' }, h('div', { class: 'ps-tri-t', title: help || '' }, title, h('span', { class: 'p-sub', text: ` (igen: ${counts.yes || 0})` })),
      h('div', { class: 'seg' }, B2B_TRI.map(([v, t]) => h('button', { type: 'button', class: (filter[key] || '') === v ? 'on' : '', text: t,
        onclick: () => { if (v) filter[key] = v; else delete filter[key]; query(); } }))));
  }

  function renderLeft() {
    if (!last) return;
    const f = last.facets;
    const presets = (B2B.state.settings.presets || []);
    const presetSel = h('select', { class: 'inp', onchange: e => {
      const p = presets.find(x => x.name === e.target.value);
      if (!p) return;
      presetName = p.name;
      filter = JSON.parse(JSON.stringify(p.filter || {}));
      if (p.group !== group) { group = p.group; renderGroups(); }
      query();
    } }, h('option', { value: '', text: presets.length ? '— mentett halmazok —' : 'nincs mentett halmaz' }), presets.map(p => h('option', { value: p.name, selected: p.name === presetName, text: `${p.name} (${p.group.replace('_', ' ')})` })));
    const search = h('input', { class: 'inp', type: 'search', placeholder: 'Név, e-mail vagy Nazon…', value: filter.query || '', spellcheck: false });
    search.addEventListener('input', debounce(() => { if (search.value.trim()) filter.query = search.value; else delete filter.query; query(); }, 250));
    const from = h('input', { class: 'inp', type: 'date', value: filter.subFrom || '', onchange: e => { if (e.target.value) filter.subFrom = e.target.value; else delete filter.subFrom; query(); } });
    const to = h('input', { class: 'inp', type: 'date', value: filter.subTo || '', onchange: e => { if (e.target.value) filter.subTo = e.target.value; else delete filter.subTo; query(); } });
    const active = left.contains(document.activeElement) && document.activeElement.type === 'search';
    left.replaceChildren(
      h('div', { class: 'ps-presets' }, presetSel,
        h('button', { class: 'btn btn-ghost btn-sm', title: 'A jelenlegi feltételek mentése névvel', onclick: savePreset }, icon('save', 15)),
        presetName ? h('button', { class: 'btn btn-ghost btn-sm btn-danger', title: 'A mentett halmaz törlése', onclick: deletePreset }, icon('trash', 15)) : null),
      h('div', { class: 'ps-searchrow' }, search),
      facet('Területi képviselő', 'reps', 'repsNot', f.reps, { open: true }),
      facet('Besorolás', 'levels', 'levelsNot', f.levels, { open: true }),
      facet('Partnerbolt / horgászbolt', 'shops', 'shopsNot', f.shops),
      facet('Megye', 'counties', 'countiesNot', f.counties),
      tri('Bizományosok', 'commission', f.commission, 'Bizományos profil (Bizomanyos = igen)'),
      tri('Belső másolati címek', 'internal', f.internal, 'Az Energofish saját címei (Fix = igen), hogy a cég is megkapja a levelet'),
      h('div', { class: 'ps-dates' }, h('div', { class: 'ps-tri-t', text: 'Feliratkozás dátuma' }), h('div', { class: 'ps-daterow' }, from, h('span', { text: '–' }), to)),
      facet('Tulajdonság 6 (régi szűrő)', 'props', 'propsNot', f.props),
      h('button', { class: 'btn btn-ghost btn-sm ps-reset', onclick: () => { filter = {}; presetName = ''; query(); } }, icon('reset', 15), 'Minden feltétel törlése'));
    if (active) { search.focus(); search.setSelectionRange(search.value.length, search.value.length); }
  }

  function renderRight() {
    const g = b2bGroup(group);
    if (!last || !last.synced) {
      right.replaceChildren(h('div', { class: 'ps-empty' }, icon('users', 34),
        h('h3', { text: g && g.configured ? 'Még nincs letöltött partnertörzs' : 'Ehhez a célcsoporthoz nincs forrás' }),
        h('p', { text: g && g.configured ? 'Töltsd le a friss címtörzset – utána itt állíthatod össze a partnerhalmazt.' : 'Add meg a célcsoport tokenes linkjét a Források között.' }),
        g && g.configured
          ? h('button', { class: 'btn btn-primary', dataset: { busy: 'Letöltés…' }, onclick: e => busy(e.currentTarget, doSync) }, icon('download'), 'Partnertörzs letöltése')
          : h('button', { class: 'btn btn-primary', onclick: () => openSources(group, () => { b2bLoadState().then(() => { renderGroups(); query(); }); }) }, icon('link'), 'Források…')));
      loadBtn.disabled = true;
      loadBtn.replaceChildren(icon('check'), 'Betöltés');
      return;
    }
    const tabs = [['', `Halmaz (${last.count})`], ['excluded', `Egyenként kizárva (${last.excluded})`], ['inactive', `Leiratkozott / inaktív (${last.inactive})`]];
    if (last.noMail) tabs.push(['nomail', `Nem kaphat levelet (${last.noMail})`]);
    const rows = last.rows.map(r => {
      const ex = new Set(filter.exclude || []);
      const act = tab === '' ? h('button', { class: 'btn btn-ghost btn-sm', title: 'Kizárás ebből a halmazból', onclick: () => { ex.add(r.email); filter.exclude = Array.from(ex); query(); } }, icon('x', 15))
        : tab === 'excluded' ? h('button', { class: 'btn btn-ghost btn-sm', title: 'Visszavétel a halmazba', onclick: () => { ex.delete(r.email); filter.exclude = Array.from(ex); if (!filter.exclude.length) delete filter.exclude; query(); } }, icon('plus', 15))
          : null;
      const tags = [];
      if (r.fix) tags.push(h('span', { class: 'pill dark', text: 'belső' }));
      if (r.commission) tags.push(h('span', { class: 'pill warn', text: 'bizományos' }));
      if (r.noToken) tags.push(h('span', { class: 'pill muted', title: 'Nincs érvényes partner-token („Torolt”)', text: 'token nélkül' }));
      return h('tr', null,
        h('td', null, h('div', { class: 'p-name', text: r.name }), h('div', { class: 'p-sub', text: `${r.email} · ${r.nazon}` })),
        h('td', null, h('div', { text: r.rep || '–' }), h('div', { class: 'p-sub', text: r.repMono })),
        h('td', null, h('div', { text: r.level }), h('div', { class: 'p-sub', text: r.shop || '' })),
        h('td', null, h('div', { text: r.county }), h('div', { class: 'p-sub', text: r.subscribed ? r.subscribed.slice(0, 10).replace(/-/g, '.') + '.' : '' })),
        h('td', null, tab === 'inactive' ? h('div', { class: 'p-sub', text: r.inactivated ? 'inaktív: ' + fmtTime(r.inactivated) : '' }) : tab === 'nomail' ? h('div', { class: 'p-sub err', text: r.noMail }) : tags),
        h('td', { class: 'c-act' }, act));
    });
    loadBtn.disabled = !last.count;
    loadBtn.replaceChildren(icon('check'), `Betöltés a hírlevélhez (${last.count} partner)`);
    right.replaceChildren(
      h('div', { class: 'ps-count' },
        h('div', null, h('b', { text: last.count }), h('span', { text: ` partner a halmazban · ${last.mailable} levelezhetőből · ${last.reps} képviselő` })),
        h('div', { class: 'ps-summary', text: last.summary })),
      h('div', { class: 'ps-tabs' }, tabs.map(([v, t]) => h('button', { type: 'button', class: tab === v ? 'on' : '', text: t, onclick: () => { tab = v; query(); } }))),
      h('div', { class: 'ps-table' }, h('table', { class: 'pt' },
        h('thead', null, h('tr', null, h('th', { text: 'Partner' }), h('th', { text: 'Képviselő' }), h('th', { text: 'Besorolás' }), h('th', { text: 'Megye · feliratkozás' }), h('th'), h('th'))),
        h('tbody', null, rows.length ? rows : h('tr', null, h('td', { colspan: 6, class: 'empty', text: tab === '' ? 'A feltételeknek egy partner sem felel meg.' : 'Nincs ilyen partner.' })))),
        last.more ? h('div', { class: 'pk-more', text: 'Csak az első 400 partner látszik – a betöltés mindet tartalmazza.' }) : null),
      h('details', { class: 'ps-opts' },
        h('summary', null, icon('pen', 15), 'A levélbe kerülő adatok', h('span', { class: 'p-sub', text: ' – megszólítás, nevek írásmódja, képviselő neve és fotója' })),
        h('div', { class: 'ps-opts-body' },
          h('div', { class: 'ps-map' }, 'A partner adatai a Tartalom változóiba kerülnek: ',
            ['{nev}', '{ceg}', '{email}', '{kepviselo}', '{terulet}', '{megye}', '{nazon}', '{besorolas}'].map((t, i) => [i ? ' ' : '', h('span', { class: 'tok', text: t })]),
            ' … (a {nev} és a {ceg} is a partner neve, a {terulet} a partner megyéje).'),
          h('div', { class: 'ps-tri' }, h('div', { class: 'ps-tri-t', text: 'Megszólítás' }),
            h('div', { class: 'seg' }, B2B_GREET.map(([v, t, tip]) => h('button', { type: 'button', title: tip, class: (options.greeting || 'name') === v ? 'on' : '', text: t,
              onclick: () => { options.greeting = v; renderRight(); } })))),
          h('label', { class: 'ps-check' }, h('input', { type: 'checkbox', checked: !options.keepCaps, onchange: e => { options.keepCaps = !e.target.checked; } }),
            'Csupa nagybetűs nevek olvasható írásmóddal (JDB HUNGARY ZRT. → JDB Hungary Zrt.)'),
          h('label', { class: 'ps-check' }, h('input', { type: 'checkbox', checked: !options.keepRepSuffix, onchange: e => { options.keepRepSuffix = !e.target.checked; } }),
            'A képviselő nevéből a „ - Energofish Kft.” utótag elhagyása'),
          h('button', { class: 'btn btn-outline btn-sm', onclick: () => openRepPhotos(group, o => { options = o; }) }, icon('user', 15), 'Képviselő-fotók…'))));
  }

  async function doSync(force) {
    const r = await api('/api/b2b/sync', { group, force: !!force });
    B2B.state = r;
    renderGroups();
    if (r.failed) { b2bSyncFailed(r, group, () => { b2bLoadState().then(() => { renderGroups(); query(); }); }); return; }
    toast('Partnertörzs letöltve: ' + b2bLogText(r.result.log), 'ok', { timeout: 8000 });
    query();
  }

  async function savePreset() {
    const v = await formModal('Partnerhalmaz mentése', [{ k: 'name', label: 'A halmaz neve (pl. „Szél Zsófia boltjai – bizományosok nélkül”)', value: presetName }], 'Mentés');
    if (!v || !v.name.trim()) return;
    try {
      const r = await api('/api/b2b/presets', { action: 'save', name: v.name.trim(), group, filter });
      B2B.state.settings.presets = r.presets;
      presetName = v.name.trim();
      renderLeft();
      toast('A halmaz elmentve.', 'ok');
    } catch (e) { toast(e.message, 'err'); }
  }

  async function deletePreset() {
    if (!await confirmBox('Mentett halmaz törlése', `A(z) „${presetName}” mentett halmaz törlődik (a partnerek nem).`, 'Törlés', true)) return;
    try {
      const r = await api('/api/b2b/presets', { action: 'delete', name: presetName });
      B2B.state.settings.presets = r.presets;
      presetName = '';
      renderLeft();
    } catch (e) { toast(e.message, 'err'); }
  }

  async function doLoad() {
    const r = await api('/api/b2b/load', { group, filter, name: presetName, options });
    done();
    applyExcel(r, true);
    const errs = S.issues.partners.filter(i => i.level === 'error').length;
    toast(`Betöltve: ${S.excel.partners.length} partner a(z) ${group.replace('_', ' ')} partnertörzsből` + (errs ? ` · ${errs} hiba` : ''), errs ? 'warn' : 'ok');
  }

  groupSel.addEventListener('change', () => { group = groupSel.value; filter = {}; presetName = ''; tab = ''; renderGroups(); query(); });
  const jsonInput = h('input', { type: 'file', accept: '.json,application/json', hidden: true, onchange: async () => {
    const f = jsonInput.files[0];
    jsonInput.value = '';
    if (!f) return;
    try { if (await b2bUpload(f, group, false)) { renderGroups(); query(); } else { renderGroups(); } } catch (e) { toast(e.message, 'err'); }
  } });
  const bg = h('div', { class: 'modal-bg' },
    h('div', { class: 'modal pset', role: 'dialog', 'aria-label': 'Partnerhalmaz összeállítása' },
      h('div', { class: 'mh' }, h('span', { text: 'Partnerhalmaz a B2B partnertörzsből' }), h('button', { class: 'x', title: 'Bezárás', onclick: done }, icon('x', 18))),
      h('div', { class: 'ps-top' },
        h('div', { class: 'ps-gwrap' }, h('span', { class: 'label-caps', text: 'Célcsoport' }), groupSel),
        syncInfo, h('div', { class: 'grow' }),
        h('button', { class: 'btn btn-outline btn-sm', dataset: { busy: 'Letöltés…' }, onclick: e => busy(e.currentTarget, () => doSync(false)) }, icon('refresh', 15), 'Frissítés most'),
        h('button', { class: 'btn btn-ghost btn-sm', title: 'A böngészőben letöltött export (JSON) betöltése ehhez a célcsoporthoz', onclick: () => jsonInput.click() }, icon('upload', 15), 'JSON-fájl…'),
        h('button', { class: 'btn btn-ghost btn-sm', onclick: () => openSources(group, () => { b2bLoadState().then(() => { renderGroups(); query(); }); }) }, icon('link', 15), 'Források…')),
      h('div', { class: 'ps-body' }, left, right), jsonInput,
      h('div', { class: 'mf pk-foot' },
        h('div', { class: 'p-sub', text: 'Csak az aktív (feliratkozott) partnerek választhatók. Egy szemponton belül bármelyik, a szempontok között mindegyik feltételnek teljesülnie kell.' }),
        h('div', { class: 'grow' }),
        h('button', { class: 'btn btn-ghost', text: 'Mégse', onclick: done }), loadBtn)));
  document.body.append(bg);
  document.addEventListener('keydown', key);
  renderGroups();
  runQuery();
}

// Források (tokenes linkek) – a tokenek nem látszanak, csak kitakarva. A panel a Források
// ablakban és a Beállításokban is ugyanaz.
function sourcesPanel(st, focusGroup) {
  const list = h('div', { class: 'src-list' });
  const paste = h('textarea', { class: 'inp', rows: 3, spellcheck: false, placeholder: 'B2B HU: https://energofish.hu/admintool/webgalamb_mod.php?action=export&token=…\nB2B SK: https://…' });
  const render = () => {
    list.replaceChildren(...B2B.state.groups.map(g => {
      const inp = h('input', { class: 'inp', type: 'password', autocomplete: 'off', spellcheck: false, placeholder: g.configured ? 'új token vagy link (felülírja)' : 'token vagy tokenes link' });
      const save = async () => {
        if (!inp.value.trim()) return;
        try { B2B.state = await api('/api/b2b/sources', { group: g.id, value: inp.value }); inp.value = ''; render(); toast(`${g.label}: forrás mentve.`, 'ok'); } catch (e) { toast(e.message, 'err'); }
      };
      inp.addEventListener('keydown', e => { if (e.key === 'Enter') save(); });
      return h('div', { class: 'src-row' + (g.id === focusGroup ? ' focus' : '') },
        h('div', { class: 'src-g' }, h('b', { text: g.label }), h('span', { class: 'p-sub', text: g.country })),
        h('div', { class: 'src-st' }, g.configured
          ? h('span', { class: 'pill ok', title: g.origin === 'env' ? `Környezeti változóból: ${g.env}` : 'Titkosítva mentve ezen a gépen', text: (g.origin === 'env' ? 'környezeti változó · ' : '') + g.masked })
          : h('span', { class: 'pill muted', text: 'nincs megadva' })),
        h('div', { class: 'src-in' }, inp, h('button', { class: 'btn btn-outline btn-sm', text: 'Mentés', onclick: save }),
          g.configured && g.origin !== 'env' ? h('button', { class: 'btn btn-ghost btn-sm btn-danger', title: 'A mentett forrás törlése', onclick: async () => {
            if (!await confirmBox('Forrás törlése', `A(z) ${g.label} tokenje törlődik erről a gépről. A már letöltött partnertörzs megmarad.`, 'Törlés', true)) return;
            try { B2B.state = await api('/api/b2b/sources', { group: g.id, remove: true }); render(); } catch (e) { toast(e.message, 'err'); }
          } }, icon('trash', 15)) : null));
    }));
  };
  render();
  const el = h('div', { class: 'src-panel' },
    h('div', { class: 'note ' + (st.protected ? 'ok' : 'warn') }, icon('shield'),
      h('div', { text: st.protected
        ? 'A tokenek titkosan, a Windows-felhasználódhoz kötve (DPAPI) tárolódnak ezen a gépen; a program sehol nem írja ki őket – itt is csak az első és utolsó 4 karakter látszik. Környezeti változóból (pl. WEBGALAMB_TOKEN_B2B_HU) is megadhatók.'
        : 'Ezen a rendszeren nincs Windows-titkosítás: a tokenek csak a felhasználó által olvasható fájlba kerülnek. Környezeti változóból (pl. WEBGALAMB_TOKEN_B2B_HU) is megadhatók.' })),
    h('div', { class: 'label-caps', style: { margin: '12px 0 6px' }, text: 'Több forrás egyszerre (beillesztés vagy egy .txt behúzása az ablakba)' }),
    paste,
    h('div', { class: 'btn-row', style: { margin: '8px 0 12px' } }, h('button', { class: 'btn btn-outline btn-sm', onclick: async () => {
      try {
        B2B.state = await api('/api/b2b/sources', { text: paste.value });
        paste.value = '';
        render();
        toast('Mentve: ' + (B2B.state.saved || []).join(', '), 'ok');
      } catch (e) { toast(e.message, 'err'); }
    } }, icon('save', 15), 'Felismerés és mentés')),
    list);
  return { el, paste };
}

async function openSources(focusGroup, after) {
  let st;
  try { st = await b2bLoadState(); } catch (e) { toast(e.message, 'err'); return; }
  const panel = sourcesPanel(st, focusGroup);
  const done = () => { bg.remove(); if (after) after(); };
  const bg = h('div', { class: 'modal-bg' },
    h('div', { class: 'modal wide src', role: 'dialog' },
      h('div', { class: 'mh', text: 'Partnertörzs-források' }),
      h('div', { class: 'mb' }, panel.el),
      h('div', { class: 'mf' }, h('button', { class: 'btn btn-primary', text: 'Kész', onclick: done }))));
  document.body.append(bg);
  panel.paste.focus();
}

// Alaphelyzet: a betöltött termékek és/vagy partnerek törlése.
async function resetData(products, partners) {
  const parts = [];
  if (products) parts.push(`A hírlevél ${S.state.products.length} terméke törlődik (az Excel-fájl és a cikktörzs nem változik).`);
  if (partners && S.excel) parts.push(S.excel.source === 'b2b'
    ? 'A betöltött partnerhalmaz kikerül (a partnertörzs, a források és a mentett halmazok megmaradnak).'
    : 'A betöltött partnerlista kikerül (az Excel-fájl nem változik).');
  if (!parts.length) { toast('Nincs betöltött termék vagy partner.', 'info'); return; }
  if (!await confirmBox('Alaphelyzet', parts.join('\n'), 'Törlés', true)) return;
  try {
    if (products) {
      S.state.products = [];
      await syncNow();
      renderProducts();
    }
    if (partners && S.excel) {
      const r = await api('/api/excel/close');
      S.excel = null;
      setIssues(r.issues);
      S.sel = new Set(); S.pv = -1;
      refreshTokenNote();
      renderPreviewSelect();
    }
    renderData();
    renderSteps();
    refreshPreview();
    toast(products && partners ? 'A termékek és a partnerek törölve.' : products ? 'A termékek törölve.' : 'A partnerlista bezárva.', 'ok');
  } catch (e) { toast(e.message, 'err'); }
}

// Beállítások: import Excel neve és mappája, cikktörzs címe, partnertörzs-források, alaphelyzet.
async function openSettings(focus) {
  let st, b2;
  try { [st, b2] = await Promise.all([api('/api/settings'), b2bLoadState()]); } catch (e) { toast(e.message, 'err'); return; }
  S.import = st.import;
  const sec = (id, ic, title, sub, ...body) => h('section', { class: 'set-sec', id: 'set-' + id },
    h('div', { class: 'set-h' }, h('div', { class: 'group-ic' }, icon(ic, 18)), h('div', null, h('h3', { text: title }), h('p', { class: 'p-sub', text: sub }))),
    body);

  // import Excel
  const nameInp = h('input', { class: 'inp', value: st.import.excelName, spellcheck: false, placeholder: st.import.defaultName });
  const dirTxt = h('div', { class: 'set-path' });
  const status = h('div', { class: 'set-status' });
  const renderImport = im => {
    S.import = im;
    nameInp.value = im.excelName;
    dirTxt.textContent = im.excelDir + (im.excelDir === im.defaultDir ? '  (a program mappája)' : '');
    status.replaceChildren(im.exists
      ? h('span', { class: 'pill ok', text: `megvan · módosítva: ${fmtTime(im.modTime)}` })
      : h('span', { class: 'pill warn', text: 'ezen a helyen még nincs ilyen fájl' }));
    if (!S.excel) renderData();
  };
  const saveImport = async (body) => {
    try { renderImport((await api('/api/settings/save', body)).import); toast('Az import Excel beállítása mentve.', 'ok'); } catch (e) { toast(e.message, 'err'); }
  };
  nameInp.addEventListener('keydown', e => { if (e.key === 'Enter') saveImport({ excelName: nameInp.value }); });
  renderImport(st.import);
  const importSec = sec('import', 'sheet', 'Import Excel', 'Ha Excelből dolgozol, a program csak ezt a nevű fájlt olvassa be: induláskor a megadott mappából, illetve behúzva vagy tallózva bárhonnan.',
    h('div', { class: 'set-grid' },
      h('label', { class: 'label-caps', text: 'Pontos fájlnév' }),
      h('div', { class: 'set-row' }, nameInp,
        h('button', { class: 'btn btn-outline btn-sm', text: 'Mentés', onclick: () => saveImport({ excelName: nameInp.value }) }),
        h('button', { class: 'btn btn-ghost btn-sm', text: 'Alapértelmezett', title: st.import.defaultName, onclick: () => saveImport({ excelName: '' }) })),
      h('label', { class: 'label-caps', text: 'Mappa' }),
      h('div', { class: 'set-row' }, dirTxt,
        h('button', { class: 'btn btn-outline btn-sm', onclick: async () => { try { renderImport((await api('/api/settings/folder')).import); } catch (e) { toast(e.message, 'err'); } } }, icon('folder', 15), 'Mappa…'),
        h('button', { class: 'btn btn-ghost btn-sm', text: 'A program mappája', onclick: () => saveImport({ excelDir: '' }) }),
        h('button', { class: 'btn btn-ghost btn-sm', title: 'A mappa megnyitása', onclick: () => api('/api/settings/open').catch(e => toast(e.message, 'err')) }, icon('external', 15))),
      h('label', { class: 'label-caps', text: 'Állapot' }),
      h('div', { class: 'set-row' }, status,
        h('button', { class: 'btn btn-primary btn-sm', onclick: async e => { await busy(e.currentTarget, importExcel); } }, icon('upload', 15), 'Betöltés most'))));

  // cikktörzs
  const o = feedOpts();
  const feedInp = h('input', { class: 'inp', type: 'url', value: o.url || '', placeholder: st.feedURL, spellcheck: false });
  const saveFeed = v => { o.url = v; feedInp.value = v; syncNow(); refreshFeed(true, v); toast('A cikktörzs címe mentve, a letöltés elindult.', 'ok'); };
  const feedSec = sec('feed', 'box', 'Cikktörzs (termékfeed)', 'Innen keres az „Új termék” ablak. Üresen az alapértelmezett Energofish nagyker feed.',
    h('div', { class: 'set-row' }, feedInp,
      h('button', { class: 'btn btn-outline btn-sm', text: 'Mentés', onclick: () => saveFeed(feedInp.value.trim()) }),
      h('button', { class: 'btn btn-ghost btn-sm', text: 'Alapértelmezett', onclick: () => saveFeed('') })),
    S.feed ? h('div', { class: 'p-sub', style: { marginTop: '6px' }, text: S.feed.ready ? `${S.feed.count.toLocaleString('hu-HU')} cikk · letöltve: ${fmtTime(S.feed.dataTime)}` : (S.feed.lastError || 'még nincs letöltve') }) : null);

  // partnertörzs-források
  const srcSec = sec('sources', 'link', 'B2B partnertörzs-források', 'A célcsoportok (országok) tokenes exportlinkjei. A program mellé tett partnerforrasok.txt-t indításkor automatikusan beolvassa.',
    sourcesPanel(b2, null).el);

  // alaphelyzet
  const resetSec = sec('reset', 'reset', 'Alaphelyzet', 'A betöltött adatok törlése a hírlevélből. A források, a beállítások és a közös tartalom megmaradnak.',
    h('div', { class: 'btn-row' },
      h('button', { class: 'btn btn-outline btn-sm btn-danger', onclick: () => resetData(true, false) }, icon('trash', 15), `Termékek törlése (${S.state.products.length})`),
      h('button', { class: 'btn btn-outline btn-sm btn-danger', onclick: () => resetData(false, true) }, icon('x', 15), `Partnerek törlése (${S.excel ? S.excel.partners.length : 0})`),
      h('button', { class: 'btn btn-dark btn-sm', onclick: () => resetData(true, true) }, 'Mindkettő')),
    h('div', { class: 'set-factory' },
      h('div', null, h('b', { text: 'Teljes visszaállítás alapállapotba' }),
        h('div', { class: 'p-sub', text: 'Minden beállítás az első indításkori értékre áll (a régiekről másolat készül). Ha a program el sem indul rendesen: EnergofishHirlevel.exe -alaphelyzet' })),
      h('button', { class: 'btn btn-dark btn-sm', onclick: () => openFactoryReset() }, icon('reset', 15), 'Visszaállítás…')));

  const nav = h('div', { class: 'set-nav' }, [['import', 'Import Excel'], ['feed', 'Cikktörzs'], ['sources', 'Partnertörzs-források'], ['reset', 'Alaphelyzet']].map(([id, t]) =>
    h('button', { type: 'button', text: t, onclick: () => { const el = document.getElementById('set-' + id); if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' }); } })));
  const done = () => { document.removeEventListener('keydown', key); bg.remove(); };
  const key = e => { if (e.key === 'Escape' && !document.querySelector('.modal-bg + .modal-bg')) done(); };
  const bg = h('div', { class: 'modal-bg', onclick: e => { if (e.target === bg) done(); } },
    h('div', { class: 'modal settings', role: 'dialog' },
      h('div', { class: 'mh' }, h('span', null, icon('gear', 18), ' Beállítások'), h('button', { class: 'x', title: 'Bezárás', onclick: done }, icon('x', 18))),
      nav,
      h('div', { class: 'mb' }, importSec, feedSec, srcSec, resetSec),
      h('div', { class: 'mf' }, h('button', { class: 'btn btn-primary', text: 'Kész', onclick: done }))));
  document.body.append(bg);
  document.addEventListener('keydown', key);
  if (focus) { const el = document.getElementById('set-' + focus); if (el) el.scrollIntoView({ block: 'start' }); }
}

// Képviselő-fotók (a partnertörzsben nincs kép): monogramonként egy kép link.
async function openRepPhotos(group, after) {
  let reps, st;
  try { [reps, st] = await Promise.all([api('/api/b2b/reps', { group }), b2bLoadState()]); } catch (e) { toast(e.message, 'err'); return; }
  const options = JSON.parse(JSON.stringify(st.settings.options || {}));
  options.repPhotos = options.repPhotos || {};
  const rows = reps.reps.map(r => {
    const prev = avatar(r.name, r.photo, 40);
    const inp = h('input', { class: 'inp', type: 'url', value: options.repPhotos[r.mono] || '', placeholder: 'https://… (négyzetes, min. 128×128)', spellcheck: false });
    inp.addEventListener('input', debounce(() => { options.repPhotos[r.mono] = inp.value.trim(); prev.replaceWith(Object.assign(avatar(r.name, inp.value.trim(), 40), {})); }, 400));
    return h('div', { class: 'rep-row' }, prev,
      h('div', { class: 'rep-who' }, h('b', { text: r.name || r.mono }), h('div', { class: 'p-sub', text: `${r.mono} · ${r.partners} partner · ${r.email || ''}` })), inp);
  });
  const done = () => bg.remove();
  const save = async () => {
    try {
      const r = await api('/api/b2b/options', options);
      if (r.excel) applyExcel(r, true);
      toast('A képviselő-fotók mentve.', 'ok');
      done();
      if (after) after(options);
    } catch (e) { toast(e.message, 'err'); }
  };
  const bg = h('div', { class: 'modal-bg' },
    h('div', { class: 'modal wide', role: 'dialog' },
      h('div', { class: 'mh', text: 'Képviselő-fotók' }),
      h('div', { class: 'mb' },
        h('p', { class: 'card-sub', text: 'A partnertörzsben nincs képviselő-fotó: itt adhatod meg monogramonként (https:// kép link). Fotó nélkül a levélben a monogram jelenik meg.' }),
        rows.length ? h('div', { class: 'rep-list' }, rows) : h('div', { class: 'note info' }, icon('info'), h('div', { text: 'Előbb töltsd le a partnertörzset.' }))),
      h('div', { class: 'mf' }, h('button', { class: 'btn btn-ghost', text: 'Mégse', onclick: done }), h('button', { class: 'btn btn-primary', text: 'Mentés', onclick: save }))));
  document.body.append(bg);
}

async function init() {
  buildShell();
  const boot = h('div', { class: 'boot' }, h('span', { class: 'spin dark' }), h('span', { text: 'A program betöltése…' }));
  const ed = $('.editor-inner');
  if (ed) ed.prepend(boot);
  let d;
  try {
    d = await api('/api/init');
  } catch (e) {
    $('.editor-inner').replaceChildren(h('div', { class: 'note err' }, icon('error'), h('div', { text: 'A program nem tudott elindulni: ' + e.message })));
    return;
  }
  Object.assign(S, {
    fields: d.fields, groups: d.groups, templates: d.templates, tokens: d.tokens, defaults: d.defaults,
    sample: d.sample, state: d.state, excel: d.excel, mode: d.mode, config: d.config, feedURL: d.feedURL, import: d.import,
  });
  setInterval(() => api('/api/heartbeat').catch(() => {}), 20000);
  try {
    S.state = S.state || {};
    if (!Array.isArray(S.state.products)) S.state.products = [];
    if (!S.state.content) S.state.content = {};
    if (!S.state.output) S.state.output = {};
    if (!S.state.feed) S.state.feed = {};
    S.excel = normalizeExcel(S.excel);
    if (d.feed) setFeedStatus(d.feed);
    for (const n of d.notices || []) toast(n.text, n.kind, { timeout: 14000 });
    setIssues(d.issues);
    S.sel = new Set(S.excel ? S.excel.partners.map((_, i) => i) : []);
    S.pv = S.excel && S.excel.partners.length ? 0 : -1;
    renderTemplateSeg();
    renderData();
    renderContent();
    renderProducts();
    renderPreviewSelect();
    const saved = ls('tab');
    setTab(STEPS.some(s => s.id === saved) ? saved : (S.excel ? 'tartalom' : 'adatok'));
    boot.remove();
    refreshPreview();
  } catch (e) {
    showRecovery(e);
  }
}

// A felület nem tudott felépülni (pl. sérült betöltött adat): helyreállítási lehetőségek.
function showRecovery(err) {
  console.error(err);
  const box = h('div', { class: 'recovery card card-pad' },
    h('div', { class: 'card-title' }, icon('alert'), 'A program felülete nem tudott betöltődni'),
    h('p', { class: 'card-sub', text: 'Valószínűleg egy hibás betöltött adat (pl. rossz Excel) okozza. Az alábbi gombokkal helyreállíthatod; a hozzáadott sablonok, a letöltött partnertörzs és a cikktörzs megmaradnak.' }),
    h('div', { class: 'note err' }, icon('error'), h('div', { text: 'Hiba: ' + (err && err.message ? err.message : String(err)) })),
    h('div', { class: 'btn-row' },
      h('button', { class: 'btn btn-primary', onclick: async e => {
        await busy(e.currentTarget, async () => {
          try { await api('/api/excel/close'); await api('/api/state', Object.assign({}, S.state || {}, { products: [] })); } catch (x) { /* tovább */ }
          location.reload();
        });
      } }, icon('trash', 16), 'Betöltött partnerek és termékek törlése'),
      h('button', { class: 'btn btn-dark', onclick: () => openFactoryReset() }, icon('reset', 16), 'Teljes visszaállítás alapállapotba…')));
  const ed = $('.editor-inner') || document.body;
  ed.replaceChildren(box);
}

// A betöltött Excel/partnerhalmaz adatainak egységesítése (null helyett üres lista).
function normalizeExcel(ex) {
  if (!ex) return null;
  for (const k of ['partners', 'products', 'partnerColumns', 'productColumns', 'issues', 'sheets', 'tokens']) if (!Array.isArray(ex[k])) ex[k] = [];
  return ex;
}

// Teljes visszaállítás alapállapotba (a menüből is elérhető, akkor is, ha a felület nem töltődött be).
async function openFactoryReset() {
  const src = h('input', { type: 'checkbox' });
  const b2b = h('input', { type: 'checkbox' });
  const ok = await new Promise(resolve => {
    const done = v => { bg.remove(); resolve(v); };
    const bg = h('div', { class: 'modal-bg' }, h('div', { class: 'modal', role: 'dialog' },
      h('div', { class: 'mh', text: 'Visszaállítás alapállapotba' }),
      h('div', { class: 'mb' },
        h('p', { style: { margin: '0 0 10px' }, text: 'A program úgy indul újra, mintha most használnád először: a közös tartalom a tervezői mintára, a termékek a mintatermékekre állnak vissza, a betöltött partnerlista, az import Excel beállítása, a kimeneti mappa és a cikktörzs címe alapértékre kerül.' }),
        h('p', { class: 'p-sub', style: { margin: '0 0 12px' }, text: 'A régi beállításokról másolat készül (beallitasok-mentes-….json a beállítások mappájában). A hozzáadott sablonok, a letöltött partnertörzs és a cikktörzs megmaradnak.' }),
        h('label', { class: 'ps-check' }, src, 'A partnertörzs-források (linkek) is törlődjenek'),
        h('label', { class: 'ps-check', style: { marginTop: '6px' } }, b2b, 'A mentett partnerhalmazok és képviselő-fotók is törlődjenek')),
      h('div', { class: 'mf' }, h('button', { class: 'btn btn-ghost', text: 'Mégse', onclick: () => done(false) }), h('button', { class: 'btn btn-dark', text: 'Visszaállítás', onclick: () => done(true) }))));
    document.body.append(bg);
  });
  if (!ok) return;
  try {
    clearTimeout(S.syncTimer);
    const r = await api('/api/settings/reset', { sources: src.checked, b2b: b2b.checked });
    try { localStorage.clear(); } catch (e) { /* nincs */ }
    toast('A program alapállapotba állt' + (r.backup ? ' (a régi beállítások másolata elkészült).' : '.'), 'ok');
    setTimeout(() => location.reload(), 600);
  } catch (e) { toast(e.message, 'err'); }
}

init();
