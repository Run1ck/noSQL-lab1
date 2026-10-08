import { api, esc, formatDateTime, formatLongDay, mountShell, requireUser, statusBadge, toast, toastError } from "./common.js";

mountShell("requests");

const user = requireUser();
const listEl = document.getElementById("requests");
const created = Number(new URLSearchParams(location.search).get("created")) || 0;
let names = new Map();

function itemsHTML(items) {
  return items
    .map((it) => `<span class="pill"><b>${esc(names.get(it.service_id) || it.service_id)}</b> · ${esc(formatLongDay(it.date))}</span>`)
    .join("");
}

function render(requests) {
  if (!requests.length) {
    listEl.innerHTML = `
      <section class="panel glass empty">
        <div class="big">📭</div>
        <p>Заявок пока нет</p>
        <a class="btn primary" href="/">Перейти в каталог</a>
      </section>`;
    return;
  }

  listEl.innerHTML = requests
    .map(
      (r) => `
      <article class="req glass" id="req-${r.id}" ${r.id === created ? 'style="border-color: var(--accent-2)"' : ""}>
        <div class="req-head">
          <span class="title">Заявка #${r.id}</span>
          ${statusBadge(r.status)}
          <span class="spacer"></span>
          <span class="faint small">${esc(formatDateTime(r.created_at))}</span>
        </div>
        <div class="req-items">${itemsHTML(r.items)}</div>
        ${r.comment ? `<div class="note">Комментарий администратора: ${esc(r.comment)}</div>` : ""}
        ${r.processed_at && r.status !== "cancelled" ? `<div class="faint small">Решение принято ${esc(formatDateTime(r.processed_at))}</div>` : ""}
        ${r.status === "new" ? `<div class="row"><span class="spacer"></span><button class="btn sm bad" data-cancel="${r.id}">Отменить заявку</button></div>` : ""}
      </article>`,
    )
    .join("");
}

async function load() {
  try {
    const [requests, services] = await Promise.all([api("GET", "/requests"), api("GET", "/services")]);
    names = new Map(services.map((s) => [s.id, s.name]));
    render(requests);
  } catch (err) {
    toastError(err);
    listEl.innerHTML = "";
  }
}

listEl.addEventListener("click", async (event) => {
  const btn = event.target.closest("[data-cancel]");
  if (!btn || !confirm("Отменить заявку?")) {
    return;
  }
  btn.disabled = true;
  try {
    await api("POST", `/requests/${btn.dataset.cancel}/cancel`);
    toast("Заявка отменена", "ok");
    await load();
  } catch (err) {
    toastError(err);
    btn.disabled = false;
  }
});

if (user) {
  if (created) {
    toast(`Заявка #${created} отправлена администратору`, "ok");
    history.replaceState(null, "", "/requests.html");
  }
  await load();
}
