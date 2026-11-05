// Core Platform administration UI.
//
// Deliberately dependency-free and build-free: one HTML file, one stylesheet,
// one module. A reviewer can open it, the browser runs it, and nothing has to be
// installed. The API is the product; this is a thin, honest client.
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

const STORAGE_API = 'core.apiBase';
const DEFAULT_API = 'http://localhost:18080';

const state = {
  apiBase: localStorage.getItem(STORAGE_API) || DEFAULT_API,
  token: null,
  user: null,
  orgs: [],
  orgId: null,
  org: null,
  perms: new Set(),
  tab: 'general',
  authMode: 'login',
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

// ------------------------------------------------------------------- API call

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
    throw new ApiError(err.message || res.statusText, res.status, err.code || '');
  }
  return payload && 'data' in payload ? payload.data : payload;
}

async function refreshSession() {
  try {
    const data = await request('/api/v1/auth/refresh', { method: 'POST', body: {}, retry: false });
    state.token = data.access_token;
    if (data.user) state.user = data.user;
    return true;
  } catch {
    state.token = null;
    return false;
  }
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
  $('#api-label').textContent = state.apiBase;
  $('#api-base').value = state.apiBase;
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
  $('#api-label').textContent = state.apiBase;
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

  $('#api-base').addEventListener('change', (event) => {
    const value = event.target.value.trim().replace(/\/$/, '');
    state.apiBase = value || DEFAULT_API;
    localStorage.setItem(STORAGE_API, state.apiBase);
    $('#api-label').textContent = state.apiBase;
  });

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
    state.token = null;
    state.user = null;
    state.orgs = [];
    state.orgId = null;
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
  await renderPanel();
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

  box.replaceChildren(
    h('span', {}, h('strong', {}, state.org.name)),
    h('span', {}, `slug: ${state.org.slug}`),
    h('span', {}, `durum: ${state.org.status}`),
    h('span', {}, `doğrulama: ${state.org.verification_level}`),
    h('span', {}, `rol: ${state.org.role}`),
  );
  box.hidden = false;
}

function bindOrgForm() {
  $('#org-select').addEventListener('change', async (event) => {
    state.orgId = event.target.value || null;
    state.org = state.orgs.find((o) => o.id === state.orgId) || null;
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
      state.tab = tab.dataset.tab;
      for (const other of $$('[data-tab]')) other.classList.toggle('active', other === tab);
      await renderPanel();
    });
  }
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
    ? h('table', { class: 'table' },
      h('thead', {}, h('tr', {},
        h('th', {}, 'Ad'), h('th', {}, 'Tür'), h('th', {}, 'Şehir'),
        h('th', {}, 'Durum'), h('th', {}, 'Public'), h('th', {}, ''))),
      h('tbody', {}, list.map((b) => h('tr', {},
        h('td', {}, b.name),
        h('td', {}, b.type),
        h('td', {}, b.city || '—'),
        h('td', {}, b.status),
        h('td', {}, b.public ? 'evet' : 'hayır'),
        h('td', {}, editable
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
          : null)))))
    : h('p', { class: 'muted' }, 'Henüz şube yok.');

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

  const rows = h('table', { class: 'table' },
    h('thead', {}, h('tr', {},
      h('th', {}, 'Kişi'), h('th', {}, 'Rol'), h('th', {}, 'Durum'), h('th', {}, ''))),
    h('tbody', {}, members.map((m) => h('tr', {},
      h('td', {}, m.user_id ? m.user_id : (m.invited_email || '—')),
      h('td', {}, m.role_key),
      h('td', {}, m.status),
      h('td', {}, editable && m.role_key !== 'owner'
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
        : null)))));

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

async function renderCatalog() {
  const orgId = state.orgId;
  const [offerings, taxonomy, categories] = await Promise.all([
    request(`/api/v1/organizations/${orgId}/offerings`),
    request('/api/v1/taxonomy'),
    request(`/api/v1/organizations/${orgId}/categories`).catch(() => []),
  ]);

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
    ? h('table', { class: 'table' },
      h('thead', {}, h('tr', {},
        h('th', {}, 'Ad'), h('th', {}, 'Kategori'), h('th', {}, 'Durum'),
        h('th', {}, 'Görünürlük'), h('th', {}, ''))),
      h('tbody', {}, offerings.map((o) => h('tr', {},
        h('td', {}, h('strong', {}, o.name), h('div', { class: 'muted small-text' }, o.slug)),
        h('td', {}, taxonomyById.get(o.taxonomy_id)?.name || '—'),
        h('td', {}, o.status),
        h('td', {}, o.visibility),
        h('td', { class: 'actions' },
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
            : null)))))
    : h('p', { class: 'muted' }, 'Henüz sunum yok. Katalog, işletmenin “ne sunduğunu” tutar; fiyat/süre/stok motora aittir.');

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

  return h('div', { class: 'stack' }, h('div', { class: 'card' }, rows), form);
}

async function renderNotifications() {
  const [list, prefs] = await Promise.all([
    request('/api/v1/notifications'),
    request('/api/v1/notifications/preferences'),
  ]);

  const items = (list.items || list || []).slice(0, 20);
  const rows = items.length
    ? h('ul', { class: 'list' }, items.map((n) => h('li', {},
      h('strong', {}, n.subject || n.template || 'Bildirim'),
      h('div', { class: 'muted small-text' }, `${n.channel} · ${fmtDate(n.created_at)}`))))
    : h('p', { class: 'muted' }, 'Gelen kutusu boş.');

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

  return h('div', { class: 'grid' },
    h('div', { class: 'card' }, h('h3', {}, 'Gelen kutusu'), rows),
    form);
}

boot();
