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

function tplShort() {
  const t = S.templates.find(t => t.id === S.state.template);
  return t ? t.short : 'v4';
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
    item('save', 'Tartalom mentése fájlba…', exportContent),
    item('open', 'Tartalom betöltése fájlból…', importContent),
    item('reset', 'Közös tartalom visszaállítása a mintára…', resetContent),
    h('hr'),
    item('download', 'Minta Excel mentése…', saveDemo),
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
  p.replaceChildren(secHead('01 / Adatforrás', 'Partnerlista Excelből',
    'Az Excel első munkalapján soronként egy partner (e-mail, név, területi képviselő), a másodikon az ajánlat termékei. Ami minden partnernél ugyanaz, azt a Tartalom lépésben adod meg.'));
  if (!S.excel) {
    p.append(dropZone(), formatCard());
    return;
  }
  p.append(fileCard(), statsRow());
  for (const i of S.issues.partners.filter(i => i.scope === 'excel')) p.append(noteFor(i));
  p.append(partnerCard());
}

function dropZone() {
  return h('div', { class: 'drop', id: 'dropZone' },
    h('div', { class: 'big-ic' }, icon('sheet', 30)),
    h('h3', { text: 'Húzd ide a partnerlista Excel fájlját' }),
    h('p', { text: 'vagy válaszd ki a gépedről (.xlsx munkafüzet)' }),
    h('div', { class: 'btn-row', style: { justifyContent: 'center' } },
      h('button', { class: 'btn btn-primary', onclick: e => busy(e.currentTarget, browseExcel) }, icon('upload'), 'Excel kiválasztása…'),
      h('button', { class: 'btn btn-outline', onclick: e => busy(e.currentTarget, saveDemo) }, icon('download'), 'Minta Excel mentése')));
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
        abs ? h('span', { class: 'p-sub', style: { marginLeft: 'auto' }, text: 'Az Excel módosítása után mentsd a fájlt, majd Újratöltés.' }) : null));
}

function statsRow() {
  const ex = S.excel;
  const blocked = S.issues.blocked || 0;
  const reps = ex.reps || 0;
  return h('div', { class: 'stats' },
    h('div', { class: 'stat accent' }, h('b', { text: ex.partners.length }), h('span', { text: 'partner' })),
    h('div', { class: 'stat' }, h('b', { text: reps }), h('span', { text: 'képviselő' })),
    h('div', { class: 'stat' }, h('b', { text: (ex.products || []).length }), h('span', { text: 'termék' })),
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
    if (!q || norm([p.name, p.email, p.company, p.repName, p.repRegion].join(' ')).includes(q)) out.push(i);
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
  const vis = visiblePartners();
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
      h('td', null, h('div', { class: 'p-name', text: p.name || '(név nélkül)' }), h('div', { class: 'p-sub', text: p.email || '– nincs e-mail –' }), p.company ? h('div', { class: 'p-sub', text: p.company }) : null),
      h('td', null, p.repName ? h('div', { class: 'rep' }, avatar(p.repName, p.repPhoto), h('div', { style: { minWidth: 0 } }, h('div', { class: 'rep-name', text: p.repName }), h('div', { class: 'rep-sub', text: p.repRegion || p.repEmail || '' }))) : h('span', { class: 'p-sub', text: '–' })),
      h('td', { class: 'c-st' }, stEl));
    return tr;
  });
  body.replaceChildren(...rows);
  if (!rows.length) body.append(h('tr', null, h('td', { colspan: 5, class: 'empty', text: 'Nincs találat.' })));
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
  S.excel = r.excel || null;
  if (r.products) S.state.products = r.products;
  setIssues(r.issues);
  S.sel = new Set(S.excel ? S.excel.partners.map((_, i) => i) : []);
  S.pv = S.excel && S.excel.partners.length ? 0 : -1;
  S.search = '';
  renderData();
  renderProducts();
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
  if (!await confirmBox('Lista bezárása', 'A partnerlista kikerül a programból (az Excel-fájl nem változik). A termékek megmaradnak.', 'Bezárás')) return;
  try {
    const r = await api('/api/excel/close');
    S.excel = null;
    setIssues(r.issues);
    S.sel = new Set(); S.pv = -1;
    renderData(); renderPreviewSelect(); refreshPreview();
  } catch (e) { toast(e.message, 'err'); }
}

async function saveDemo() {
  try {
    const r = await api('/api/demo/save');
    if (r.cancelled) return;
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
    const f = e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0];
    if (f) uploadExcel(f);
  });
}

/* ------------------------------------------------------------------ 2. Tartalom */

function renderContent() {
  const p = pane('tartalom');
  const open = new Set(ls('groups') || ['alap', 'level']);
  p.replaceChildren(
    secHead('02 / Közös tartalom', 'Ami minden partnernél ugyanaz',
      'A mezők a hírlevél blokkjait követik, fentről lefelé. A jobb oldali előnézet gépelés közben frissül.'),
    h('div', { class: 'note cream' }, icon('users'),
      h('div', null, h('b', { text: 'Partnerenként cserélődő változók: ' }),
        S.tokens.slice(0, 7).map((t, i) => [i ? ' ' : '', h('span', { class: 'tok', title: t.desc, text: t.token })]),
        '. Pl. a megszólítás „Kedves {nev}!” – a {nev} helyére minden levélben a partner neve kerül.')));
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
        h('div', { class: 'group-title' }, g.title, g.only ? h('span', { class: 'pill dark', text: 'csak ' + g.only.join(', ') }) : null),
        h('div', { class: 'group-desc', text: g.desc })),
      h('div', { class: 'group-meta' }, h('span', { class: 'gstat' }), icon('chev', 18)));
    head.lastChild.lastChild.classList.add('chev');
    const body = h('div', { class: 'group-body' },
      g.only ? h('div', { class: 'inactive-note hidden', text: 'A kiválasztott sablon ezt a blokkot nem használja, a mezők értéke megmarad.' }) : null,
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

function showTokenBar(wrap, input) {
  $$('.tokenbar').forEach(t => { if (t.parentElement !== wrap) t.remove(); });
  if ($('.tokenbar', wrap)) return;
  const isUrl = input.dataset.key && /url|Pattern$/i.test(input.dataset.key);
  const tokens = S.tokens.filter(t => t.token !== '{assets}' || isUrl);
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
  const short = tplShort();
  $$('.group').forEach(card => {
    const g = S.groups.find(x => x.id === card.dataset.group);
    const inactive = !!(g && g.only && !g.only.includes(short));
    card.classList.toggle('inactive', inactive);
    const note = $('.inactive-note', card);
    if (note) note.classList.toggle('hidden', !inactive);
  });
  for (const f of S.fields) {
    const el = $(`.field[data-key="${f.key}"]`, pane('tartalom'));
    if (el) el.classList.toggle('dim', !!(f.only && !f.only.includes(short)));
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
      h('button', { class: 'btn btn-primary btn-sm', onclick: addProduct }, icon('plus', 16), 'Új termék')),
    h('div', { id: 'pGeneral' }),
    h('div', { id: 'plist' }, list.length ? list.map(productCard) : h('div', { class: 'card empty' }, 'Még nincs termék. Tölts be Excelt Termékek munkalappal, vagy vegyél fel egyet kézzel.')));
  updateProductIssues();
  renderSteps();
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
    upd();
    return h('div', { class: 'field' + (f.half ? ' half' : ''), dataset: { k: f.k } },
      h('div', { class: 'field-top' }, h('label', null, f.label, f.req ? h('span', { class: 'req', text: '*' }) : null), counter),
      inp, h('div', { class: 'msg' }));
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

function addProduct() {
  S.state.products.push({ on: true, code: '', name: '', desc: '', price: '', deal: '', image: '', url: '', alt: '', cta: '' });
  renderProducts();
  syncSoon(0);
  const last = $('#plist').lastElementChild;
  if (last) { last.scrollIntoView({ block: 'center' }); $('input.inp', last).focus(); }
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
    h('p', { class: 'card-sub', text: 'Letölti a borító-, termék-, portré- és képviselőképeket, és ellenőrzi, hogy elérhetők-e, és megfelelő-e a méretük (borító 1200×660, termék min. 260×260, négyzetes). Internetkapcsolat kell hozzá.' }),
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
    h('div', { class: 'card card-pad' },
      h('div', { class: 'card-title' }, icon('mail'), 'Sablon'),
      h('p', { class: 'card-sub', text: 'Az előnézet feletti kapcsolóval is váltható.' }),
      h('div', { class: 'tpl-cards' }, S.templates.map(t => h('button', { class: 'tpl-card' + (t.id === S.state.template ? ' sel' : ''), onclick: () => setTemplate(t.id) },
        h('div', { class: 'mini ' + t.short }, h('i', { class: 'hd' }), h('i', { class: 'cv' }), h('i', { class: 'tx' }), h('div', { class: 'gr' }, h('i'), h('i'), h('i'), h('i'), h('i'), h('i')), h('i', { class: 'ft' })),
        h('div', { class: 'nm' }, h('span', { class: 'pill dark', text: t.short }), t.name),
        h('div', { class: 'ds', text: t.desc }))))),
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
    const r = await api('/api/generate', { only });
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

/* ------------------------------------------------------------------ előnézet */

function buildPreview() {
  const a = $('#preview');
  const devBtn = (id, ic, title) => h('button', { dataset: { dev: id }, title, onclick: () => setDevice(id) }, icon(ic, 17));
  a.append(
    h('div', { class: 'pv-top' },
      h('div', { class: 'seg', id: 'tplSeg' }),
      h('div', { class: 'grow' }),
      h('div', { class: 'seg icons', id: 'devSeg' }, devBtn('desktop', 'monitor', 'Asztali nézet'), devBtn('mobile', 'phone', 'Mobil nézet (620 px alatt 2 oszlopos rács)'))),
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
  S.device = ls('device') || 'desktop';
  setDevice(S.device, true);
  new ResizeObserver(() => fitFrame()).observe($('#pvCanvas'));
}

function renderTemplateSeg() {
  const seg = $('#tplSeg');
  seg.replaceChildren(...S.templates.map(t => h('button', { class: t.id === S.state.template ? 'on' : '', title: t.desc, onclick: () => setTemplate(t.id) },
    h('span', { class: 'sh', text: t.short.toUpperCase() }), t.name)));
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
  $('#pvStage').classList.toggle('mobile', id === 'mobile');
  if (!silent) { fitFrame(); refreshPreview(); }
}

function renderPreviewSelect() {
  const sel = $('#pvSel');
  if (!S.excel || !S.excel.partners.length) {
    sel.replaceChildren(h('option', { value: -1, text: `Minta partner – ${S.sample.name} (${S.sample.email})` }));
    sel.value = -1;
    return;
  }
  sel.replaceChildren(...S.excel.partners.map((p, i) => h('option', { value: i, text: `${i + 1}. ${p.name || '(név nélkül)'} – ${p.email || 'nincs e-mail'}${isBlocked(i) ? '  ⚠' : ''}` })));
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

function frameWidth() { return S.device === 'mobile' ? 390 : 660; }

function fitFrame() {
  const frame = $('#pvFrame'), canvas = $('#pvCanvas'), stage = $('#pvStage'), sizer = $('#pvSizer');
  if (!frame || !canvas) return;
  const W = frameWidth();
  let H = 900;
  try {
    const d = frame.contentDocument;
    if (d && d.documentElement) H = Math.max(d.documentElement.scrollHeight, d.body ? d.body.scrollHeight : 0, 300);
  } catch (e) { /* nincs hozzáférés */ }
  const avail = canvas.clientWidth - 32 - (S.device === 'mobile' ? 24 : 0);
  const s = Math.min(1, Math.max(0.3, avail / W));
  frame.style.width = W + 'px';
  frame.style.height = H + 'px';
  stage.style.width = W + 'px';
  stage.style.transform = `scale(${s})`;
  stage.style.transformOrigin = 'top left';
  const extra = S.device === 'mobile' ? 24 : 0;
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

async function init() {
  buildShell();
  let d;
  try {
    d = await api('/api/init');
  } catch (e) {
    $('.editor-inner').replaceChildren(h('div', { class: 'note err' }, icon('error'), h('div', { text: 'A program nem tudott elindulni: ' + e.message })));
    return;
  }
  Object.assign(S, {
    fields: d.fields, groups: d.groups, templates: d.templates, tokens: d.tokens, defaults: d.defaults,
    sample: d.sample, state: d.state, excel: d.excel, mode: d.mode, config: d.config,
  });
  if (!S.state.output) S.state.output = {};
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
  refreshPreview();
  setInterval(() => api('/api/heartbeat').catch(() => {}), 20000);
}

init();
