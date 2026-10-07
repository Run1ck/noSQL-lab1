import {
  KINDS,
  api,
  esc,
  formatDateTime,
  formatLongDay,
  mountShell,
  requireUser,
  statusBadge,
  toast,
  toastError,
} from "./common.js";

mountShell("admin");

const user = requireUser({ admin: true });
const requestsEl = document.getElementById("requests");
const servicesEl = document.getElementById("services");
const form = document.getElementById("service-form");
let status = "new";
let services = [];

const names = () => new Map(services.map((s) => [s.id, s.name]));

function renderRequests(requests) {
  if (!requests.length) {
    requestsEl.innerHTML = `<section class="panel glass empty"><div class="big">🎉</div>Заявок нет</section>`;
    return;
  }
  const byId = names();
  requestsEl.innerHTML = requests
    .map(
      (r) => `
      <article class="req glass">
        <div class="req-head">
          <span class="title">Заявка #${r.id}</span>
          ${statusBadge(r.status)}
          <span class="mono faint" title="${esc(r.user_id)}">${esc(r.user_id.slice(0, 8))}</span>
          <span class="spacer"></span>
          <span class="faint small">${esc(formatDateTime(r.created_at))}</span>
        </div>
        <div class="req-items">
          ${r.items
            .map((it) => `<span class="pill"><b>${esc(byId.get(it.service_id) || it.service_id)}</b> · ${esc(formatLongDay(it.date))}</span>`)
            .join("")}
        </div>
        ${r.comment ? `<div class="note">${esc(r.comment)}</div>` : ""}
        ${
          r.status === "new"
            ? `<div class="req-actions">
                 <input class="input" placeholder="Комментарий (необязательно)" data-comment="${r.id}">
                 <button class="btn ok" data-action="approve" data-id="${r.id}">Одобрить</button>
                 <button class="btn bad" data-action="reject" data-id="${r.id}">Отклонить</button>
               </div>`
            : r.processed_at
              ? `<div class="faint small">Обработана ${esc(formatDateTime(r.processed_at))}</div>`
              : ""
        }
      </article>`,
    )
    .join("");
}

async function loadRequests() {
  requestsEl.innerHTML = `<div class="skeleton"></div><div class="skeleton"></div>`;
  try {
    renderRequests(await api("GET", "/admin/requests" + (status ? `?status=${status}` : "")));
  } catch (err) {
    toastError(err);
    requestsEl.innerHTML = "";
  }
}

function renderServices() {
  document.getElementById("services-count").textContent = services.length ? `· ${services.length}` : "";
  if (!services.length) {
    servicesEl.innerHTML = `<tr><td colspan="5" class="empty">Каталог пуст — добавьте первую услугу</td></tr>`;
    return;
  }
  servicesEl.innerHTML = services
    .map(
      (s) => `
      <tr>
        <td class="mono">${esc(s.id)}</td>
        <td>${esc(s.name)}</td>
        <td><span class="row"><span class="kind-dot kind-${esc(s.kind)}"></span>${esc(KINDS[s.kind] || s.kind)}</span></td>
        <td>${s.active ? '<span class="badge approved">Активна</span>' : '<span class="badge inactive">Скрыта</span>'}</td>
        <td style="text-align:right;white-space:nowrap">
          <button class="btn sm" data-edit="${esc(s.id)}">Изменить</button>
          <button class="btn sm ghost" data-toggle="${esc(s.id)}">${s.active ? "Скрыть" : "Показать"}</button>
        </td>
      </tr>`,
    )
    .join("");
}

async function loadServices() {
  try {
    services = await api("GET", "/admin/services");
    renderServices();
  } catch (err) {
    toastError(err);
  }
}

async function saveService(s) {
  const saved = await api("PUT", `/admin/services/${encodeURIComponent(s.id)}`, {
    name: s.name,
    kind: s.kind,
    active: s.active,
  });
  await loadServices();
  return saved;
}

function fillForm(s) {
  form.elements.id.value = s?.id || "";
  form.elements.id.readOnly = Boolean(s);
  form.elements.name.value = s?.name || "";
  form.elements.kind.value = s?.kind || "room";
  form.elements.active.checked = s ? s.active : true;
  document.getElementById("service-form-title").textContent = s ? `Редактирование · ${s.id}` : "Новая услуга";
}

requestsEl.addEventListener("click", async (event) => {
  const btn = event.target.closest("[data-action]");
  if (!btn) {
    return;
  }
  const { action, id } = btn.dataset;
  const comment = requestsEl.querySelector(`[data-comment="${id}"]`)?.value.trim() || "";
  btn.closest(".req-actions").querySelectorAll("button").forEach((b) => (b.disabled = true));
  try {
    await api("POST", `/admin/requests/${id}/${action}`, { comment });
    toast(action === "approve" ? `Заявка #${id} одобрена` : `Заявка #${id} отклонена`, "ok");
  } catch (err) {
    toastError(err);
  }
  await loadRequests();
});

document.getElementById("status-filter").addEventListener("click", (event) => {
  const chip = event.target.closest(".chip");
  if (!chip) {
    return;
  }
  status = chip.dataset.status;
  document.querySelectorAll("#status-filter .chip").forEach((c) => c.classList.toggle("active", c === chip));
  loadRequests();
});

document.getElementById("tabs").addEventListener("click", (event) => {
  const chip = event.target.closest(".chip");
  if (!chip) {
    return;
  }
  document.querySelectorAll("#tabs .chip").forEach((c) => c.classList.toggle("active", c === chip));
  document.getElementById("tab-requests").hidden = chip.dataset.tab !== "requests";
  document.getElementById("tab-services").hidden = chip.dataset.tab !== "services";
});

servicesEl.addEventListener("click", async (event) => {
  const edit = event.target.closest("[data-edit]");
  if (edit) {
    fillForm(services.find((s) => s.id === edit.dataset.edit));
    form.scrollIntoView({ behavior: "smooth", block: "center" });
    return;
  }
  const toggle = event.target.closest("[data-toggle]");
  if (toggle) {
    const s = services.find((x) => x.id === toggle.dataset.toggle);
    toggle.disabled = true;
    try {
      await saveService({ ...s, active: !s.active });
      toast(s.active ? "Услуга скрыта из каталога" : "Услуга снова в каталоге", "ok");
    } catch (err) {
      toastError(err);
      toggle.disabled = false;
    }
  }
});

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const s = {
    id: form.elements.id.value.trim(),
    name: form.elements.name.value.trim(),
    kind: form.elements.kind.value,
    active: form.elements.active.checked,
  };
  if (!s.id || !s.name) {
    toast("Укажите ID и название", "bad");
    return;
  }
  try {
    await saveService(s);
    toast(`Услуга «${s.name}» сохранена`, "ok");
    fillForm(null);
  } catch (err) {
    toastError(err);
  }
});

document.getElementById("service-reset").addEventListener("click", (event) => {
  event.preventDefault();
  fillForm(null);
});

if (user) {
  await loadServices();
  await loadRequests();
}
