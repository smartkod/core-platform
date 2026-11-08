// Core Platform administration UI.
//
// Deliberately dependency-free and build-free: one HTML file, one stylesheet,
// one module. A reviewer can open it, the browser runs it, and nothing has to be
// installed. The API is the product; this is a thin, honest client.
//
// EDITING NOTE: this file is UTF-8 without a BOM and contains Turkish text.
// Windows PowerShell 5.1 decodes a BOM-less file as the ANSI code page, so
// `Get-Content | Set-Content` silently turns "Şehir" into mojibake. Edit it with
// a UTF-8-aware tool, or explicitly: ReadAllText(path, UTF8) / WriteAllText with
// UTF8Encoding($false).
//
// Auth transport: the access token stays in memory (never localStorage, so an
// XSS bug cannot exfiltrate a long-lived credential) while the refresh token
// lives in an HttpOnly cookie the page cannot read. On reload the session is
// restored by calling /auth/refresh, which works because the browser sends that
// cookie automatically.
//
// Every write carries the X-CSRF-Token header copied from the readable `csrf`
// cookie, which is the double-submit check the API expects for cookie-authenticated
// requests.

// The API address is a deployment fact, not a user setting. A field that lets a
// visitor point the panel at an arbitrary server would also let a phishing page
// borrow this origin's cookies against an attacker-controlled backend, so it is
// derived instead: same host, API port. A deployment with a different topology
// sets window.CORE_API_BASE in its own HTML shell.
const DEFAULT_API = window.CORE_API_BASE
  || `${window.location.protocol}//${window.location.hostname}:18080`;

const state = {
  apiBase: DEFAULT_API,
  token: null,
  user: null,
  orgs: [],
  orgId: null,
  org: null,
  perms: new Set(),
  tab: 'general',
  authMode: 'login',
  // Offsets for the paginated panels. Reset when the tab or organization
  // changes, so a user does not land on page 3 of a list they just switched to.
  catalogOffset: 0,
  notificationsOffset: 0,
};

class ApiError extends Error {
  constructor(message, status, code) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

// ---------------------------------------------------------------- DOM helpers

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

/**
 * Builds an element. Text children are inserted as text nodes, never as HTML:
 * everything rendered here comes from the API or from another user, so using
 * innerHTML would turn a business name into an XSS vector.
 */
function h(tag, props = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(props)) {
    if (value === undefined || value === null || value === false) continue;
    if (key === 'class') node.className = value;
    else if (key === 'dataset') Object.assign(node.dataset, value);
    else if (key.startsWith('on') && typeof value === 'function') {
      node.addEventListener(key.slice(2).toLowerCase(), value);
    } else if (key === 'checked' || key === 'disabled' || key === 'required' || key === 'selected') {
      node[key] = Boolean(value);
    } else if (key === 'value') {
      node.value = value;
    } else {
      node.setAttribute(key, value);
    }
  }
  for (const child of children.flat()) {
    if (child === null || child === undefined || child === false) continue;
    node.append(child instanceof Node ? child : document.createTextNode(String(child)));
  }
  return node;
}

function field(labelText, input) {
  return h('label', {}, labelText, input);
}

function toast(message, kind = 'error') {
  const box = $('#toast');
  box.textContent = message;
  box.className = `toast ${kind}`;
  box.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => { box.hidden = true; }, 4200);
}

function cookie(name) {
  const prefix = `${name}=`;
  for (const part of document.cookie.split(';')) {
    const trimmed = part.trim();
    if (trimmed.startsWith(prefix)) return decodeURIComponent(trimmed.slice(prefix.length));
  }
  return '';
}

function fmtDate(value) {
  if (!value) return '—';
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString('tr-TR');
}

// A membership record carries an id, not a name: the API deliberately keeps the
// account projection out of the tenant payload. So the list shows what it can
// honestly show — "Siz" for the signed-in account, the address for an
// invitation, and a short reference for anyone else — instead of dumping a raw
// UUID, which tells a reader nothing and looks like a bug.
function personCell(m) {
  if (m.user_id && state.user && m.user_id === state.user.id) {
    return h('span', {}, h('strong', {}, 'Siz'), h('span', { class: 'muted small-text' }, ` · ${m.invited_email || ''}`.trimEnd()));
  }
  if (m.invited_email) {
    return h('span', {}, m.invited_email);
  }
  return h('span', { class: 'mono muted', title: m.user_id || '' },
    m.user_id ? `#${m.user_id.slice(-8)}` : '—');
}

const ROLE_LABELS = {
  owner: 'Sahip',
  manager: 'Yönetici',
  staff: 'Personel',
  member: 'Üye',
};

const STATUS_LABELS = {
  active: 'Etkin',
  invited: 'Davetli',
  removed: 'Kaldırıldı',
  inactive: 'Pasif',
  draft: 'Taslak',
  published: 'Yayında',
  archived: 'Arşivde',
};

function roleLabel(key) { return ROLE_LABELS[key] || key; }
function statusLabel(key) { return STATUS_LABELS[key] || key; }

// A status is a small fact, so it gets a badge rather than a bare word: a scan
// down the column then works without reading.
function badge(key) {
  const kind = { published: 'ok', active: 'ok', draft: 'warn', invited: 'warn', archived: 'muted', removed: 'danger' }[key] || '';
  return h('span', { class: `badge ${kind}` }, statusLabel(key));
}

// ------------------------------------------------------------------- API call

// A single in-flight refresh. Two requests that both see 401 must share one
// refresh call, not make two.
//
// This is not an optimisation. The API rotates the refresh token on every use
// and treats a second presentation of the old one as theft: it revokes the whole
// token family. So two parallel refreshes do not merely waste a request — the
// second one logs the user out. The browser makes parallel requests all the
// time (a tab switch renders several panels), so without this guard the session
// would drop at random under normal use.
let refreshInFlight = null;

async function request(path, { method = 'GET', body, retry = true } = {}) {
  const headers = { 'X-Client-Type': 'web' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  const csrf = cookie('csrf');
  if (csrf) headers['X-CSRF-Token'] = csrf;

  let res;
  try {
    res = await fetch(state.apiBase + path, {
      method,
      headers,
      credentials: 'include',
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(`API'ye ulaşılamadı (${state.apiBase}). Çalışıyor mu?`, 0, 'NETWORK');
  }

  // An expired access token is the normal case after a break; refresh once and
  // replay. Retrying more than once would hide a broken session behind a loop.
  if (res.status === 401 && retry && state.token) {
    if (await refreshSession()) return request(path, { method, body, retry: false });
  }

  const text = await res.text();
  let payload = null;
  if (text) {
    try { payload = JSON.parse(text); } catch { payload = null; }
  }

  if (!res.ok) {
    const err = payload && payload.error ? payload.error : {};
    if (res.status === 401 && state.token) clearSession();
    throw new ApiError(err.message || res.statusText, res.status, err.code || '');
  }
  return payload && 'data' in payload ? payload.data : payload;
}

// refreshSession returns the shared in-flight refresh when one exists.
function refreshSession() {
  if (!refreshInFlight) {
    refreshInFlight = doRefresh().finally(() => { refreshInFlight = null; });
  }
  return refreshInFlight;
}

async function doRefresh() {
  try {
    const data = await request('/api/v1/auth/refresh', { method: 'POST', body: {}, retry: false });
    state.token = data.access_token;
    if (data.user) state.user = data.user;
    return true;
  } catch {
    clearSession();
    return false;
  }
}

// clearSession forgets the in-memory credentials. It does not call the API:
// a session that has already been rejected must not trigger another round trip.
function clearSession() {
  state.token = null;
  state.user = null;
  state.orgs = [];
  state.orgId = null;
  state.org = null;
  state.perms = new Set();
}

// Users cannot mutate permissions; this only decides which buttons to show.
// The server still refuses anything the caller may not do.
function can(action) {
  return state.perms.has(action);
}

async function loadPermissions() {
  state.perms = new Set();
  if (!state.orgId) return;
  try {
    const data = await request(
      `/api/v1/authz/me/permissions?scope=organization&scope_id=${encodeURIComponent(state.orgId)}`);
    for (const key of data.permissions || []) state.perms.add(key);
  } catch {
    // A user without a membership simply gets an empty set; every guarded
    // action then renders as unavailable instead of failing on click.
  }
}

// --------------------------------------------------------------- boot + auth

async function boot() {
  bindAuthForm();
  bindOrgForm();
  bindTabs();

  if (await refreshSession()) {
    await enterApp();
  } else {
    showAuth();
  }
}

function showAuth() {
  $('#auth-view').hidden = false;
  $('#app-view').hidden = true;
  $('#logout').hidden = true;
  $('#whoami').textContent = '';
}

function setAuthMode(mode) {
  state.authMode = mode;
  for (const tab of $$('[data-auth]')) tab.classList.toggle('active', tab.dataset.auth === mode);
  for (const node of $$('.register-only')) node.hidden = mode !== 'register';
  $('#auth-submit').textContent = mode === 'login' ? 'Giriş yap' : 'Hesap oluştur';
  $('#auth-hint').textContent = mode === 'login'
    ? 'Hesabınız yoksa “Kayıt” sekmesine geçin.'
    : 'Parola en az 10 karakter olmalı.';
  const password = $('#auth-form [name=password]');
  password.autocomplete = mode === 'login' ? 'current-password' : 'new-password';
}

function bindAuthForm() {
  for (const tab of $$('[data-auth]')) {
    tab.addEventListener('click', () => setAuthMode(tab.dataset.auth));
  }

  $('#auth-form').addEventListener('submit', async (event) => {
    event.preventDefault();
    const form = new FormData(event.target);
    const email = String(form.get('email') || '').trim();
    const password = String(form.get('password') || '');
    const button = $('#auth-submit');
    button.disabled = true;

    try {
      if (state.authMode === 'register') {
        await request('/api/v1/auth/register', {
          method: 'POST',
          body: { email, password, name: String(form.get('name') || '').trim() },
        });
        toast('Hesap oluşturuldu. Şimdi giriş yapabilirsiniz.', 'ok');
        setAuthMode('login');
        return;
      }

      const data = await request('/api/v1/auth/login', { method: 'POST', body: { email, password } });
      state.token = data.access_token;
      state.user = data.user;
      event.target.reset();
      await enterApp();
    } catch (err) {
      toast(err.message || 'İşlem başarısız');
    } finally {
      button.disabled = false;
    }
  });

  $('#logout').addEventListener('click', async () => {
    try { await request('/api/v1/auth/logout', { method: 'POST', body: {} }); } catch { /* already gone */ }
    clearSession();
    showAuth();
  });
}

async function enterApp() {
  if (!state.user) {
    const me = await request('/api/v1/me');
    state.user = me;
  }
  $('#auth-view').hidden = true;
  $('#app-view').hidden = false;
  $('#logout').hidden = false;
  $('#whoami').textContent = state.user.name || state.user.email;

  await loadOrganizations();

  // Deep link: #katalog, #subeler, ... . The panel is a single page, so a tab
  // that cannot be linked is a tab a user cannot bookmark or send to a
  // colleague.
  applyHashTab();
  await renderPanel();
}

const TAB_HASHES = {
  general: '#genel',
  branches: '#subeler',
  members: '#ekip',
  catalog: '#katalog',
  notifications: '#bildirimler',
};

function applyHashTab() {
  const hash = location.hash.toLowerCase();
  for (const [tab, value] of Object.entries(TAB_HASHES)) {
    if (value === hash) state.tab = tab;
  }
  syncTabUI();
}

function selectTab(tab) {
  state.tab = tab;
  state.catalogOffset = 0;
  state.notificationsOffset = 0;
  const hash = TAB_HASHES[tab];
  if (hash && location.hash !== hash) history.replaceState(null, '', hash);
  syncTabUI();
}

function syncTabUI() {
  for (const button of $$('[data-tab]')) {
    const active = button.dataset.tab === state.tab;
    button.classList.toggle('active', active);
    button.setAttribute('aria-selected', String(active));
    button.tabIndex = active ? 0 : -1;
  }
}

// ---------------------------------------------------------------- org switch

async function loadOrganizations() {
  const assignments = await request('/api/v1/authz/me/assignments');
  const byOrg = new Map();
  for (const a of assignments) {
    if (a.scope !== 'organization' || !a.scope_id) continue;
    byOrg.set(a.scope_id, a.role_key);
  }

  const orgs = [];
  for (const [id, role] of byOrg) {
    try {
      const org = await request(`/api/v1/organizations/${id}`);
      orgs.push({ ...org, role });
    } catch {
      // A grant that outlived its organization is not an error worth blocking
      // the whole page for; the entry is simply skipped.
    }
  }
  state.orgs = orgs;

  if (!orgs.some((o) => o.id === state.orgId)) state.orgId = orgs.length ? orgs[0].id : null;

  const select = $('#org-select');
  select.replaceChildren();
  if (!orgs.length) {
    select.append(h('option', { value: '' }, 'Henüz işletme yok'));
  } else {
    for (const org of orgs) {
      select.append(h('option', { value: org.id, selected: org.id === state.orgId },
        `${org.name} (${org.role})`));
    }
  }
  select.disabled = !orgs.length;

  const hasOrg = Boolean(state.orgId);
  $('#tabs').hidden = !hasOrg;
  $('#org-meta').hidden = true;

  if (hasOrg) {
    state.org = orgs.find((o) => o.id === state.orgId) || null;
    await loadPermissions();
    showOrgMeta();
  } else {
    state.perms = new Set();
    state.org = null;
  }
}

function showOrgMeta() {
  const box = $('#org-meta');
  if (!state.org) { box.hidden = true; return; }

  const meta = (label, value) => h('span', { class: 'meta-item' },
    h('span', { class: 'meta-label' }, label),
    h('span', {}, value));

  box.replaceChildren(
    h('span', { class: 'meta-name' }, state.org.name),
    meta('Adres', `/${state.org.slug}`),
    meta('Durum', statusLabel(state.org.status)),
    meta('Doğrulama', { none: 'Yok', basic: 'Temel', verified: 'Doğrulanmış' }[state.org.verification_level] || state.org.verification_level),
    meta('Rolünüz', roleLabel(state.org.role)),
  );
  box.hidden = false;
}

function bindOrgForm() {
  $('#org-select').addEventListener('change', async (event) => {
    state.orgId = event.target.value || null;
    state.org = state.orgs.find((o) => o.id === state.orgId) || null;
    state.catalogOffset = 0;
    state.notificationsOffset = 0;
    await loadPermissions();
    showOrgMeta();
    await renderPanel();
  });

  const form = $('#new-org-form');
  const toggle = (open) => {
    form.hidden = !open;
    if (open) form.querySelector('[name=name]').focus();
  };

  $('#new-org-toggle').addEventListener('click', () => toggle(form.hidden));
  $('#new-org-cancel').addEventListener('click', () => { form.reset(); toggle(false); });

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const data = new FormData(form);
    try {
      const org = await request('/api/v1/organizations', {
        method: 'POST',
        body: {
          name: String(data.get('name') || '').trim(),
          slug: String(data.get('slug') || '').trim().toLowerCase(),
        },
      });
      form.reset();
      toggle(false);
      state.orgId = org.id;
      await loadOrganizations();
      await renderPanel();
      toast('İşletme oluşturuldu.', 'ok');
    } catch (err) {
      toast(err.message || 'İşletme oluşturulamadı');
    }
  });
}

function bindTabs() {
  for (const tab of $$('[data-tab]')) {
    tab.addEventListener('click', async () => {
      selectTab(tab.dataset.tab);
      await renderPanel();
      $('#panel').focus();
    });
  }

  // The back button must move between sections, not leave the app.
  window.addEventListener('hashchange', async () => {
    applyHashTab();
    await renderPanel();
  });
}

// -------------------------------------------------------------------- panels

async function renderPanel() {
  const panel = $('#panel');
  if (!state.orgId) {
    panel.replaceChildren(h('div', { class: 'card muted' },
      'Başlamak için bir işletme oluşturun. İşletmeyi oluşturan hesap otomatik olarak sahibi olur.'));
    return;
  }
  panel.replaceChildren(h('div', { class: 'loading' }, 'Yükleniyor…'));

  const render = {
    general: renderGeneral,
    branches: renderBranches,
    members: renderMembers,
    catalog: renderCatalog,
    notifications: renderNotifications,
  }[state.tab];

  try {
    panel.replaceChildren(await render());
  } catch (err) {
    panel.replaceChildren(h('div', { class: 'card' },
      h('p', { class: 'error' }, err.message || 'Yüklenemedi'),
      err.status === 403 ? h('p', { class: 'hint' }, 'Bu işletmede bu işlem için yetkiniz yok.') : null));
  }
}

function renderGeneral() {
  const org = state.org;
  const editable = can('organization.manage');

  const form = h('form', { class: 'card stack' },
    h('div', { class: 'row' },
      field('Ad', h('input', { name: 'name', value: org.name, disabled: !editable })),
      field('Slug (değiştirilemez)', h('input', { value: org.slug, disabled: true }))),
    h('div', { class: 'row' },
      field('E-posta', h('input', { name: 'email', type: 'email', value: org.email || '', disabled: !editable })),
      field('Telefon', h('input', { name: 'phone', value: org.phone || '', disabled: !editable }))),
    h('div', { class: 'row' },
      field('Web', h('input', { name: 'website', value: org.website || '', disabled: !editable })),
      field('Dil', h('input', { name: 'locale', value: org.locale || 'tr', disabled: !editable }))),
    field('Açıklama', h('textarea', { name: 'description', rows: 3, disabled: !editable }, org.description || '')),
    editable ? h('div', { class: 'row' }, h('button', { class: 'btn primary', type: 'submit' }, 'Kaydet')) : null,
  );

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const data = new FormData(form);
    try {
      const updated = await request(`/api/v1/organizations/${org.id}`, {
        method: 'PATCH',
        body: {
          name: String(data.get('name') || ''),
          description: String(data.get('description') || ''),
          email: String(data.get('email') || ''),
          phone: String(data.get('phone') || ''),
          website: String(data.get('website') || ''),
          locale: String(data.get('locale') || ''),
        },
      });
      Object.assign(state.org, updated);
      Object.assign(state.orgs.find((o) => o.id === org.id) || {}, updated);
      showOrgMeta();
      toast('Kaydedildi.', 'ok');
    } catch (err) {
      toast(err.message || 'Kaydedilemedi');
    }
  });

  const permissions = h('div', { class: 'card' },
    h('h3', {}, 'Bu işletmedeki izinleriniz'),
    h('div', { class: 'chips' },
      state.perms.size
        ? [...state.perms].sort().map((p) => h('span', { class: 'chip' }, p))
        : h('span', { class: 'muted' }, 'Atanmış izin yok')),
    h('p', { class: 'hint' }, 'Sunucu her isteği bu izinlere göre ayrıca denetler; burada yalnızca arayüz gösterilir.'));

  return h('div', { class: 'grid' }, form, permissions);
}

async function renderBranches() {
  const orgId = state.orgId;
  const list = await request(`/api/v1/organizations/${orgId}/branches`);
  const editable = can('org.branch.manage');

  const rows = list.length
    ? h('div', { class: 'table-wrap' },
      h('table', { class: 'table' },
        h('thead', {}, h('tr', {},
          h('th', {}, 'Ad'), h('th', {}, 'Tür'), h('th', {}, 'Şehir'),
          h('th', {}, 'Durum'), h('th', {}, 'Herkese açık'), h('th', {}, ''))),
        h('tbody', {}, list.map((b) => h('tr', {},
          h('td', {}, b.name),
          h('td', {}, b.type === 'physical' ? 'Fiziksel' : b.type === 'online' ? 'Online' : 'Hizmet alanı'),
          h('td', {}, b.city || '—'),
          h('td', {}, badge(b.status)),
          h('td', {}, b.public ? 'Evet' : 'Hayır'),
          h('td', { class: 'actions' },
            h('div', { class: 'actions-inner' },
              editable
                ? h('button', {
                  class: 'btn small danger',
                  type: 'button',
                  onclick: async () => {
                    if (!confirm(`“${b.name}” şubesi silinsin mi?`)) return;
                    try {
                      await request(`/api/v1/organizations/${orgId}/branches/${b.id}`, { method: 'DELETE' });
                      toast('Şube silindi.', 'ok');
                      await renderPanel();
                    } catch (err) { toast(err.message || 'Silinemedi'); }
                  },
                }, 'Sil')
                : null)))))))
    : h('div', { class: 'empty' },
      h('strong', {}, 'Henüz şube yok'),
      'İlk şubenizi ekleyerek başlayın. Adres bilgisi herkese açık olabilir.');

  const form = editable
    ? h('form', { class: 'card stack' },
      h('div', { class: 'row' },
        field('Ad', h('input', { name: 'name', required: true, placeholder: 'Merkez' })),
        field('Tür', h('select', { name: 'type' },
          h('option', { value: 'physical' }, 'Fiziksel'),
          h('option', { value: 'service_area' }, 'Hizmet alanı'),
          h('option', { value: 'online' }, 'Online')))),
      h('div', { class: 'row' },
        field('Şehir', h('input', { name: 'city' })),
        field('İlçe', h('input', { name: 'district' })),
        field('Telefon', h('input', { name: 'phone' }))),
      field('Adres', h('input', { name: 'address_line' })),
      h('div', { class: 'row' },
        h('label', { class: 'check' }, h('input', { type: 'checkbox', name: 'public', checked: true }), 'Adres herkese açık'),
        h('button', { class: 'btn primary', type: 'submit' }, 'Şube ekle')))
    : null;

  if (form) {
    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      const data = new FormData(form);
      try {
        await request(`/api/v1/organizations/${orgId}/branches`, {
          method: 'POST',
          body: {
            name: String(data.get('name') || ''),
            type: String(data.get('type') || 'physical'),
            city: String(data.get('city') || ''),
            district: String(data.get('district') || ''),
            phone: String(data.get('phone') || ''),
            address_line: String(data.get('address_line') || ''),
            public: data.get('public') === 'on',
          },
        });
        toast('Şube eklendi.', 'ok');
        await renderPanel();
      } catch (err) { toast(err.message || 'Eklenemedi'); }
    });
  }

  return h('div', { class: 'stack' }, h('div', { class: 'card' }, rows), form);
}

async function renderMembers() {
  const orgId = state.orgId;
  const members = await request(`/api/v1/organizations/${orgId}/members`);
  const editable = can('org.member.manage');

  const rows = h('div', { class: 'table-wrap' },
    h('table', { class: 'table' },
      h('thead', {}, h('tr', {},
        h('th', {}, 'Kişi'), h('th', {}, 'Rol'), h('th', {}, 'Durum'), h('th', {}, ''))),
      h('tbody', {}, members.map((m) => h('tr', {},
        h('td', {}, personCell(m)),
        h('td', {}, roleLabel(m.role_key)),
        h('td', {}, badge(m.status)),
        h('td', { class: 'actions' },
          h('div', { class: 'actions-inner' },
            editable && m.role_key !== 'owner'
              ? h('button', {
                class: 'btn small danger',
                type: 'button',
                onclick: async () => {
                  if (!confirm('Üyelik kaldırılsın mı?')) return;
                  try {
                    await request(`/api/v1/organizations/${orgId}/members/${m.id}`, { method: 'DELETE' });
                    toast('Üyelik kaldırıldı.', 'ok');
                    await renderPanel();
                  } catch (err) { toast(err.message || 'Kaldırılamadı'); }
                },
              }, 'Kaldır')
              : null)))))));

  const form = editable
    ? h('form', { class: 'card stack' },
      h('div', { class: 'row' },
        field('E-posta', h('input', { name: 'email', type: 'email', required: true })),
        field('Rol', h('select', { name: 'role_key' },
          h('option', { value: 'manager' }, 'Yönetici'),
          h('option', { value: 'staff' }, 'Personel'),
          h('option', { value: 'member' }, 'Üye')))),
      h('p', { class: 'hint' }, 'Davet, e-posta adresine bağlıdır: yalnızca o adrese sahip hesap kabul edebilir.'),
      h('div', { class: 'row' }, h('button', { class: 'btn primary', type: 'submit' }, 'Davet gönder')))
    : null;

  if (form) {
    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      const data = new FormData(form);
      try {
        await request(`/api/v1/organizations/${orgId}/members`, {
          method: 'POST',
          body: {
            email: String(data.get('email') || ''),
            role_key: String(data.get('role_key') || 'member'),
          },
        });
        toast('Davet oluşturuldu.', 'ok');
        await renderPanel();
      } catch (err) { toast(err.message || 'Davet gönderilemedi'); }
    });
  }

  return h('div', { class: 'stack' }, h('div', { class: 'card' }, rows), form);
}

// requestPage reads a paginated endpoint, keeping the envelope.
//
// A list endpoint answers {data: [...], total, limit, offset}; `request` returns
// only `data`, which is right for most calls but silently drops the total. A
// screen that then renders "the list" shows the first page and claims it is all
// of it, so a business with 40 offerings would only ever see 20 and have no
// reason to suspect the rest exist.
async function requestPage(path, { limit = 20, offset = 0 } = {}) {
  const sep = path.includes('?') ? '&' : '?';
  const res = await fetch(`${state.apiBase}${path}${sep}limit=${limit}&offset=${offset}`, {
    headers: pageHeaders(),
    credentials: 'include',
  });

  if (res.status === 401 && state.token && await refreshSession()) {
    return requestPage(path, { limit, offset });
  }

  const payload = await res.json().catch(() => null);
  if (!res.ok) {
    const err = (payload && payload.error) || {};
    if (res.status === 401 && state.token) clearSession();
    throw new ApiError(err.message || res.statusText, res.status, err.code || '');
  }

  return {
    items: Array.isArray(payload.data) ? payload.data : (payload.data?.items || []),
    total: payload.total ?? (payload.data?.total ?? 0),
    limit: payload.limit ?? limit,
    offset: payload.offset ?? offset,
  };
}

function pageHeaders() {
  const headers = { 'X-Client-Type': 'web' };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  const csrf = cookie('csrf');
  if (csrf) headers['X-CSRF-Token'] = csrf;
  return headers;
}

// pager renders "1–20 / 43" with previous/next, or nothing when one page holds
// everything. Hidden rather than absent when empty: a control that appears and
// disappears as data changes makes the layout jump.
function pager(page, onGoto) {
  const from = page.total === 0 ? 0 : page.offset + 1;
  const to = Math.min(page.offset + page.items.length, page.total);
  const canPrev = page.offset > 0;
  const canNext = page.offset + page.limit < page.total;

  return h('div', { class: 'pager' },
    h('span', { class: 'muted small-text' }, `${from}–${to} / ${page.total}`),
    h('div', { class: 'row', style: 'gap:8px' },
      h('button', {
        class: 'btn small', type: 'button', disabled: !canPrev,
        onclick: () => onGoto(Math.max(0, page.offset - page.limit)),
      }, 'Önceki'),
      h('button', {
        class: 'btn small', type: 'button', disabled: !canNext,
        onclick: () => onGoto(page.offset + page.limit),
      }, 'Sonraki')));
}

async function renderCatalog() {
  const orgId = state.orgId;
  const offset = state.catalogOffset || 0;
  const [page, taxonomy, categories] = await Promise.all([
    requestPage(`/api/v1/organizations/${orgId}/offerings`, { offset }),
    request('/api/v1/taxonomy'),
    request(`/api/v1/organizations/${orgId}/categories`).catch(() => []),
  ]);
  const offerings = page.items;

  const editable = can('catalog.manage');
  const publishable = can('catalog.publish');
  const taxonomyById = new Map(taxonomy.map((t) => [t.id, t]));
  for (const c of categories) taxonomyById.set(c.id, c);

  const statusButtons = (o) => {
    if (!publishable) return null;
    const move = async (status) => {
      try {
        await request(`/api/v1/organizations/${orgId}/offerings/${o.id}/status`, {
          method: 'POST', body: { status },
        });
        toast(`Durum: ${status}`, 'ok');
        await renderPanel();
      } catch (err) { toast(err.message || 'Durum değiştirilemedi'); }
    };
    const buttons = [];
    if (o.status !== 'published' && o.status !== 'archived') {
      buttons.push(h('button', { class: 'btn small', type: 'button', onclick: () => move('published') }, 'Yayınla'));
    }
    if (o.status !== 'archived') {
      buttons.push(h('button', { class: 'btn small ghost', type: 'button', onclick: () => move('archived') }, 'Arşivle'));
    }
    return buttons;
  };

  const rows = offerings.length
    ? h('div', { class: 'table-wrap' },
      h('table', { class: 'table' },
        h('thead', {}, h('tr', {},
          h('th', {}, 'Ad'), h('th', {}, 'Kategori'), h('th', {}, 'Durum'),
          h('th', {}, 'Görünürlük'), h('th', {}, ''))),
        h('tbody', {}, offerings.map((o) => h('tr', {},
        h('td', {}, h('strong', {}, o.name), h('div', { class: 'muted small-text' }, o.slug)),
        h('td', {}, taxonomyById.get(o.taxonomy_id)?.name || '—'),
        h('td', {}, badge(o.status)),
        h('td', {}, o.visibility === 'public' ? 'Genel' : 'Özel'),
        h('td', { class: 'actions' },
          h('div', { class: 'actions-inner' },
            ...(statusButtons(o) || []),
            editable && o.status !== 'published'
              ? h('button', {
                class: 'btn small danger',
                type: 'button',
                onclick: async () => {
                  if (!confirm(`“${o.name}” silinsin mi?`)) return;
                  try {
                    await request(`/api/v1/organizations/${orgId}/offerings/${o.id}`, { method: 'DELETE' });
                    toast('Sunum silindi.', 'ok');
                    await renderPanel();
                  } catch (err) { toast(err.message || 'Silinemedi'); }
                },
              }, 'Sil')
              : null)))))))
    : h('div', { class: 'empty' },
      h('strong', {}, 'Henüz sunum yok'),
      'Katalog, işletmenin “ne sunduğunu” tutar; fiyat, süre ve stok motora aittir.');

  const options = taxonomy.map((t) => h('option', { value: t.id }, t.name));

  const form = editable
    ? h('form', { class: 'card stack' },
      h('div', { class: 'row' },
        field('Ad', h('input', { name: 'name', required: true, placeholder: '60 dakikalık seans' })),
        field('Slug', h('input', { name: 'slug', required: true, placeholder: '60-dk-seans' }))),
      h('div', { class: 'row' },
        field('Kategori', h('select', { name: 'taxonomy_id', required: true }, options)),
        field('Hizmet modu', h('select', { name: 'service_mode' },
          h('option', { value: 'on_site' }, 'Yerinde'),
          h('option', { value: 'online' }, 'Online'),
          h('option', { value: 'at_customer' }, 'Müşteride'),
          h('option', { value: 'hybrid' }, 'Karma')))),
      field('Özet', h('input', { name: 'summary', maxlength: 300 })),
      h('div', { class: 'row' }, h('button', { class: 'btn primary', type: 'submit' }, 'Sunum ekle')))
    : h('p', { class: 'hint' },
      'Katalog düzenleme yetkiniz yok. İşletme yöneticinizden catalog.manage izni isteyin.');

  if (editable) {
    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      const data = new FormData(form);
      try {
        await request(`/api/v1/organizations/${orgId}/offerings`, {
          method: 'POST',
          body: {
            name: String(data.get('name') || ''),
            slug: String(data.get('slug') || '').toLowerCase(),
            taxonomy_id: String(data.get('taxonomy_id') || ''),
            service_mode: String(data.get('service_mode') || 'on_site'),
            summary: String(data.get('summary') || ''),
          },
        });
        toast('Sunum eklendi (taslak).', 'ok');
        await renderPanel();
      } catch (err) { toast(err.message || 'Eklenemedi'); }
    });
  }

  const body = h('div', { class: 'card' }, rows);
  if (page.total > page.limit) {
    body.append(pager(page, async (next) => {
      state.catalogOffset = next;
      await renderPanel();
    }));
  }

  return h('div', { class: 'stack' }, body, form);
}

async function renderNotifications() {
  const offset = state.notificationsOffset || 0;
  const [page, prefs] = await Promise.all([
    requestPage('/api/v1/notifications', { offset }),
    request('/api/v1/notifications/preferences'),
  ]);

  const items = page.items;
  const rows = items.length
    ? h('ul', { class: 'list' }, items.map((n) => h('li', {},
      h('strong', {}, n.subject || n.template || 'Bildirim'),
      h('div', { class: 'muted small-text' }, `${n.channel} · ${fmtDate(n.created_at)}`))))
    : h('div', { class: 'empty' },
      h('strong', {}, 'Gelen kutusu boş'),
      'Yeni bir bildirim geldiğinde burada görünecek.');

  const inbox = h('div', { class: 'card' }, h('h2', {}, 'Gelen kutusu'), rows);
  if (page.total > page.limit) {
    inbox.append(pager(page, async (next) => {
      state.notificationsOffset = next;
      await renderPanel();
    }));
  }

  const channels = prefs.channels || {};
  const form = h('form', { class: 'card stack' },
    h('h3', {}, 'Kanal tercihleri'),
    ...['email', 'in_app', 'push'].map((ch) => h('label', { class: 'check' },
      h('input', { type: 'checkbox', name: ch, checked: Boolean(channels[ch]) }),
      ch === 'email' ? 'E-posta' : ch === 'in_app' ? 'Uygulama içi' : 'Push (rezerve)')),
    h('div', { class: 'row' }, h('button', { class: 'btn primary', type: 'submit' }, 'Tercihleri kaydet')));

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const data = new FormData(form);
    try {
      await request('/api/v1/notifications/preferences', {
        method: 'PUT',
        body: {
          email: data.get('email') === 'on',
          in_app: data.get('in_app') === 'on',
          push: data.get('push') === 'on',
        },
      });
      toast('Tercihler kaydedildi.', 'ok');
    } catch (err) { toast(err.message || 'Kaydedilemedi'); }
  });

  return h('div', { class: 'grid' }, inbox, form);
}

boot();
