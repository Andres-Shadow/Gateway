const tokenKey = "gateway_admin_token";
const state = {
  token: localStorage.getItem(tokenKey) || "",
  routes: [],
  query: "",
};

const els = {
  loginPanel: document.querySelector("#loginPanel"),
  loginForm: document.querySelector("#loginForm"),
  dashboard: document.querySelector("#dashboard"),
  logoutButton: document.querySelector("#logoutButton"),
  refreshButton: document.querySelector("#refreshButton"),
  routeForm: document.querySelector("#routeForm"),
  routesList: document.querySelector("#routesList"),
  emptyState: document.querySelector("#emptyState"),
  routeSearch: document.querySelector("#routeSearch"),
  formTitle: document.querySelector("#formTitle"),
  saveRouteButton: document.querySelector("#saveRouteButton"),
  resetFormButton: document.querySelector("#resetFormButton"),
  toast: document.querySelector("#toast"),
};

const fields = [
  "routeId",
  "path",
  "targetUrl",
  "methods",
  "isActive",
  "healthCheckPath",
  "rewritePrefixFrom",
  "rewritePrefixTo",
  "requestHeadersSet",
  "requestHeadersRemove",
  "responseHeadersSet",
  "responseHeadersRemove",
  "requestBodyTransform",
  "responseBodyTransform",
].reduce((acc, id) => {
  acc[id] = document.querySelector(`#${id}`);
  return acc;
}, {});

function setAuthenticated(isAuthenticated) {
  els.loginPanel.hidden = isAuthenticated;
  els.dashboard.hidden = !isAuthenticated;
  els.logoutButton.hidden = !isAuthenticated;
  els.refreshButton.hidden = !isAuthenticated;
}

function showToast(message) {
  els.toast.textContent = message;
  els.toast.classList.add("visible");
  window.clearTimeout(showToast.timeout);
  showToast.timeout = window.setTimeout(() => els.toast.classList.remove("visible"), 2600);
}

async function api(path, options = {}) {
  const headers = new Headers(options.headers || {});
  headers.set("Content-Type", "application/json");
  if (state.token) {
    headers.set("Authorization", `Bearer ${state.token}`);
  }

  const response = await fetch(path, { ...options, headers });
  if (response.status === 401) {
    localStorage.removeItem(tokenKey);
    state.token = "";
    setAuthenticated(false);
    throw new Error("Sesión expirada o token inválido");
  }
  if (!response.ok) {
    const errorBody = await response.json().catch(() => ({}));
    throw new Error(errorBody.error || `Request failed: ${response.status}`);
  }
  if (response.status === 204) {
    return null;
  }
  return response.json();
}

async function login(event) {
  event.preventDefault();
  const body = {
    username: document.querySelector("#username").value.trim(),
    password: document.querySelector("#password").value,
  };
  const data = await api("/admin/auth/token", {
    method: "POST",
    body: JSON.stringify(body),
  });
  state.token = data.token;
  localStorage.setItem(tokenKey, state.token);
  setAuthenticated(true);
  await loadRoutes();
  showToast("Sesión iniciada");
}

async function loadRoutes() {
  state.routes = await api("/admin/routes");
  renderRoutes();
}

function renderRoutes() {
  const query = state.query.toLowerCase();
  const routes = state.routes.filter((route) => {
    return [route.path, route.target_url, route.methods, route.health_check_path]
      .filter(Boolean)
      .join(" ")
      .toLowerCase()
      .includes(query);
  });

  els.emptyState.hidden = routes.length > 0;
  els.routesList.innerHTML = routes.map(routeTemplate).join("");
}

function routeTemplate(route) {
  const healthURL = healthUrl(route);
  const activeClass = route.is_active ? "ok" : "off";
  const activeText = route.is_active ? "Activa" : "Inactiva";
  const healthButton = healthURL
    ? `<a class="button-link" href="${escapeHTML(healthURL)}" target="_blank" rel="noopener noreferrer">Health</a>`
    : `<button class="secondary" disabled>Health</button>`;

  return `
    <article class="route-row" data-route-id="${route.id}">
      <div class="route-main">
        <p class="route-path">${escapeHTML(route.path)}</p>
        <p class="target-url">${escapeHTML(route.target_url)}</p>
      </div>
      <div class="route-meta">
        <span class="badge ${activeClass}">${activeText}</span>
        ${methodBadges(route.methods)}
        ${route.health_check_path ? `<span class="badge warn">${escapeHTML(route.health_check_path)}</span>` : ""}
      </div>
      <div class="route-actions">
        ${healthButton}
        <button class="secondary" data-action="edit" data-id="${route.id}">Editar</button>
        <button class="secondary" data-action="toggle" data-id="${route.id}">
          ${route.is_active ? "Desactivar" : "Activar"}
        </button>
        <button class="danger" data-action="delete" data-id="${route.id}">Eliminar</button>
      </div>
    </article>
  `;
}

function methodBadges(methods = "") {
  return methods
    .split(",")
    .map((method) => method.trim())
    .filter(Boolean)
    .map((method) => `<span class="badge">${escapeHTML(method)}</span>`)
    .join("");
}

function healthUrl(route) {
  if (!route.target_url) {
    return "";
  }
  try {
    const url = new URL(route.target_url);
    url.pathname = route.health_check_path || "/healthz";
    url.search = "";
    url.hash = "";
    return url.toString();
  } catch {
    return "";
  }
}

function resetForm() {
  els.formTitle.textContent = "Nueva ruta";
  els.saveRouteButton.textContent = "Crear ruta";
  els.routeForm.reset();
  fields.routeId.value = "";
  fields.isActive.checked = true;
}

function editRoute(route) {
  els.formTitle.textContent = `Editar ruta #${route.id}`;
  els.saveRouteButton.textContent = "Guardar cambios";
  fields.routeId.value = route.id;
  fields.path.value = route.path || "";
  fields.targetUrl.value = route.target_url || "";
  fields.methods.value = route.methods || "";
  fields.isActive.checked = Boolean(route.is_active);
  fields.healthCheckPath.value = route.health_check_path || "";
  fields.rewritePrefixFrom.value = route.rewrite_prefix_from || "";
  fields.rewritePrefixTo.value = route.rewrite_prefix_to || "";
  fields.requestHeadersSet.value = prettyJSON(route.request_headers_set);
  fields.requestHeadersRemove.value = prettyJSON(route.request_headers_remove);
  fields.responseHeadersSet.value = prettyJSON(route.response_headers_set);
  fields.responseHeadersRemove.value = prettyJSON(route.response_headers_remove);
  fields.requestBodyTransform.value = route.request_body_transform || "";
  fields.responseBodyTransform.value = route.response_body_transform || "";
  fields.path.focus();
}

async function saveRoute(event) {
  event.preventDefault();
  const id = fields.routeId.value;
  const payload = buildPayload(Boolean(id));
  const route = await api(id ? `/admin/routes/${id}` : "/admin/routes", {
    method: id ? "PUT" : "POST",
    body: JSON.stringify(payload),
  });
  resetForm();
  await loadRoutes();
  showToast(`Ruta ${id ? "actualizada" : "creada"}: ${route.path}`);
}

function buildPayload(isUpdate) {
  const payload = {
    path: fields.path.value.trim(),
    target_url: fields.targetUrl.value.trim(),
    methods: fields.methods.value.trim(),
    is_active: fields.isActive.checked,
    health_check_path: fields.healthCheckPath.value.trim(),
    rewrite_prefix_from: fields.rewritePrefixFrom.value.trim(),
    rewrite_prefix_to: fields.rewritePrefixTo.value.trim(),
    request_body_transform: fields.requestBodyTransform.value,
    response_body_transform: fields.responseBodyTransform.value,
  };

  applyJSONField(payload, "request_headers_set", fields.requestHeadersSet.value, {});
  applyJSONField(payload, "request_headers_remove", fields.requestHeadersRemove.value, []);
  applyJSONField(payload, "response_headers_set", fields.responseHeadersSet.value, {});
  applyJSONField(payload, "response_headers_remove", fields.responseHeadersRemove.value, []);

  if (isUpdate) {
    return payload;
  }
  return payload;
}

function applyJSONField(payload, key, value, fallback) {
  const trimmed = value.trim();
  if (!trimmed) {
    payload[key] = fallback;
    return;
  }
  payload[key] = JSON.parse(trimmed);
}

async function handleRouteAction(event) {
  const button = event.target.closest("button[data-action]");
  if (!button) {
    return;
  }
  const route = state.routes.find((item) => String(item.id) === button.dataset.id);
  if (!route) {
    return;
  }

  if (button.dataset.action === "edit") {
    editRoute(route);
    return;
  }
  if (button.dataset.action === "toggle") {
    await api(`/admin/routes/${route.id}`, {
      method: "PUT",
      body: JSON.stringify({ is_active: !route.is_active }),
    });
    await loadRoutes();
    showToast(route.is_active ? "Ruta desactivada" : "Ruta activada");
    return;
  }
  if (button.dataset.action === "delete") {
    const confirmed = window.confirm(`Eliminar la ruta ${route.path}?`);
    if (!confirmed) {
      return;
    }
    await api(`/admin/routes/${route.id}`, { method: "DELETE" });
    await loadRoutes();
    showToast("Ruta eliminada");
  }
}

function prettyJSON(value) {
  if (!value) {
    return "";
  }
  try {
    const parsed = typeof value === "string" ? JSON.parse(value) : value;
    if (parsed == null || (Array.isArray(parsed) && parsed.length === 0)) {
      return "";
    }
    if (!Array.isArray(parsed) && typeof parsed === "object" && Object.keys(parsed).length === 0) {
      return "";
    }
    return JSON.stringify(parsed, null, 2);
  } catch {
    return value;
  }
}

function escapeHTML(value = "") {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function logout() {
  state.token = "";
  state.routes = [];
  localStorage.removeItem(tokenKey);
  setAuthenticated(false);
  resetForm();
}

function wireEvents() {
  els.loginForm.addEventListener("submit", (event) => login(event).catch((error) => showToast(error.message)));
  els.routeForm.addEventListener("submit", (event) => saveRoute(event).catch((error) => showToast(error.message)));
  els.logoutButton.addEventListener("click", logout);
  els.refreshButton.addEventListener("click", () => loadRoutes().catch((error) => showToast(error.message)));
  els.resetFormButton.addEventListener("click", resetForm);
  els.routesList.addEventListener("click", (event) => handleRouteAction(event).catch((error) => showToast(error.message)));
  els.routeSearch.addEventListener("input", (event) => {
    state.query = event.target.value;
    renderRoutes();
  });
}

async function init() {
  wireEvents();
  setAuthenticated(Boolean(state.token));
  if (state.token) {
    try {
      await loadRoutes();
    } catch (error) {
      showToast(error.message);
    }
  }
}

init();
