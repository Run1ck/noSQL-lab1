import {
  KINDS,
  addDays,
  api,
  esc,
  formatDay,
  formatDow,
  formatDuration,
  formatLongDay,
  getUser,
  mountShell,
  toast,
  toastError,
  todayISO,
} from "./common.js";

const DAYS = 7;

mountShell("catalog");

const user = getUser();
const today = todayISO();
const state = {
  services: [],
  kind: "",
  start: today,
  booked: new Map(),
  cart: new Set(),
  expiresIn: 0,
  maxTTL: 0,
  busy: false,
};

const servicesEl = document.getElementById("services");
const scheduleEl = document.getElementById("schedule");
const cartEl = document.getElementById("cart");
const cartCountEl = document.getElementById("cart-count");
const startInput = document.getElementById("start-date");

const key = (serviceId, date) => serviceId + "|" + date;

function visibleServices() {
  return state.services.filter((s) => !state.kind || s.kind === state.kind);
}

function serviceName(id) {
  return state.services.find((s) => s.id === id)?.name || id;
}

function renderServices() {
  const list = visibleServices();
  if (!list.length) {
    servicesEl.innerHTML = `<div class="empty"><div class="big">🗂️</div>Пока нет доступных услуг</div>`;
    return;
  }
  servicesEl.innerHTML = list
    .map(
      (s) => `
      <article class="service-card">
        <div class="kind"><span class="kind-dot kind-${esc(s.kind)}"></span>${esc(KINDS[s.kind] || s.kind)}</div>
        <div class="name">${esc(s.name)}</div>
        <div class="id">${esc(s.id)}</div>
      </article>`,
    )
    .join("");
}

function days() {
  return Array.from({ length: DAYS }, (_, i) => addDays(state.start, i));
}

function renderSchedule() {
  if (!user) {
    scheduleEl.innerHTML = `
      <div class="cta">
        <div class="spacer"><b>Войдите, чтобы увидеть занятость и бронировать</b>
        <div class="muted small">Расписание и корзина доступны после входа.</div></div>
        <a class="btn primary" href="/login.html">Войти</a>
      </div>`;
    return;
  }

  const list = visibleServices();
  if (!list.length) {
    scheduleEl.innerHTML = `<div class="empty">Нет услуг для выбранного фильтра</div>`;
    return;
  }

  const head = days()
    .map((d) => `<th class="${d === today ? "today" : ""}"><span class="dow">${esc(formatDow(d))}</span>${esc(formatDay(d))}</th>`)
    .join("");

  const rows = list
    .map((s) => {
      const cells = days()
        .map((d) => {
          const booked = state.booked.get(d)?.has(s.id);
          const inCart = state.cart.has(key(s.id, d));
          const past = d < today;
          let cls = "free";
          let label = "";
          let title = "Свободно — добавить в корзину";
          if (past) {
            cls = "past";
            title = "День прошёл";
          } else if (booked) {
            cls = "booked";
            label = "занято";
            title = "Уже забронировано";
          } else if (inCart) {
            cls = "in-cart";
            label = "✓";
            title = "В корзине — убрать";
          }
          const disabled = past || booked ? "disabled" : "";
          return `<td><button class="cell ${cls}" data-svc="${esc(s.id)}" data-date="${d}" title="${title}" ${disabled}>${label}</button></td>`;
        })
        .join("");
      return `<tr><td class="svc">${esc(s.name)}</td>${cells}</tr>`;
    })
    .join("");

  scheduleEl.innerHTML = `
    <div class="schedule-wrap">
      <table class="schedule"><thead><tr><th class="svc">Услуга</th>${head}</tr></thead><tbody>${rows}</tbody></table>
    </div>
    <div class="legend">
      <span><i class="l-free"></i>свободно</span>
      <span><i class="l-cart"></i>в корзине</span>
      <span><i class="l-booked"></i>занято</span>
    </div>`;
}

function renderCart() {
  if (!user) {
    cartEl.innerHTML = `<div class="empty"><div class="big">🛒</div>Корзина появится после входа</div>`;
    cartCountEl.textContent = "";
    return;
  }

  const items = [...state.cart]
    .map((k) => {
      const i = k.lastIndexOf("|");
      return { serviceId: k.slice(0, i), date: k.slice(i + 1) };
    })
    .sort((a, b) => a.date.localeCompare(b.date) || a.serviceId.localeCompare(b.serviceId));

  cartCountEl.textContent = items.length ? `· ${items.length}` : "";

  if (!items.length) {
    cartEl.innerHTML = `<div class="empty"><div class="big">✨</div>Отметьте свободные дни в расписании</div>`;
    return;
  }

  const ratio = state.maxTTL ? Math.min(100, (state.expiresIn / state.maxTTL) * 100) : 100;

  cartEl.innerHTML = `
    <div class="timer"><span>Истечёт через <b id="ttl">${formatDuration(state.expiresIn)}</b></span>
      <span class="bar"><i id="ttl-bar" style="width:${ratio}%"></i></span></div>
    <ul class="cart-list">
      ${items
        .map(
          (it) => `
        <li class="cart-item">
          <div class="what"><b>${esc(serviceName(it.serviceId))}</b><span>${esc(formatLongDay(it.date))}</span></div>
          <button class="btn sm icon-btn ghost" data-remove-svc="${esc(it.serviceId)}" data-remove-date="${it.date}" aria-label="Убрать">✕</button>
        </li>`,
        )
        .join("")}
    </ul>
    <button class="btn primary block" id="checkout">Оформить заявку</button>
    <button class="btn ghost block small" id="clear" style="margin-top:8px">Очистить корзину</button>`;
}

function applyCart(cart) {
  state.cart = new Set((cart?.items || []).map((it) => key(it.service_id, it.date)));
  state.expiresIn = cart?.expires_in || 0;
  state.maxTTL = state.cart.size ? Math.max(state.maxTTL, state.expiresIn) : 0;
}

function render() {
  renderServices();
  renderSchedule();
  renderCart();
}

async function loadServices() {
  try {
    state.services = await api("GET", "/services");
  } catch (err) {
    toastError(err);
    state.services = [];
  }
}

async function loadSchedule() {
  if (!user) {
    return;
  }
  try {
    const data = await api("GET", `/schedule?from=${state.start}&to=${addDays(state.start, DAYS - 1)}`);
    state.booked = new Map(data.days.map((d) => [d.date, new Set(d.booked)]));
  } catch (err) {
    toastError(err);
  }
}

async function loadCart() {
  if (!user) {
    return;
  }
  try {
    applyCart(await api("GET", "/cart"));
  } catch (err) {
    toastError(err);
  }
}

async function withBusy(fn) {
  if (state.busy) {
    return;
  }
  state.busy = true;
  try {
    await fn();
  } finally {
    state.busy = false;
    render();
  }
}

scheduleEl.addEventListener("click", (event) => {
  const cell = event.target.closest("button.cell");
  if (!cell || cell.disabled) {
    return;
  }
  const { svc, date } = cell.dataset;
  withBusy(async () => {
    try {
      if (state.cart.has(key(svc, date))) {
        applyCart(await api("DELETE", `/cart/items/${encodeURIComponent(svc)}/${date}`));
      } else {
        applyCart(await api("POST", "/cart/items", { service_id: svc, date }));
      }
    } catch (err) {
      toastError(err);
      if (err.code === "slot_booked") {
        await loadSchedule();
      }
    }
  });
});

cartEl.addEventListener("click", (event) => {
  const remove = event.target.closest("[data-remove-svc]");
  if (remove) {
    const { removeSvc, removeDate } = remove.dataset;
    withBusy(async () => {
      try {
        applyCart(await api("DELETE", `/cart/items/${encodeURIComponent(removeSvc)}/${removeDate}`));
      } catch (err) {
        toastError(err);
      }
    });
    return;
  }

  if (event.target.closest("#clear")) {
    withBusy(async () => {
      try {
        await api("DELETE", "/cart");
        applyCart(null);
      } catch (err) {
        toastError(err);
      }
    });
    return;
  }

  if (event.target.closest("#checkout")) {
    withBusy(async () => {
      try {
        const req = await api("POST", "/requests");
        location.href = "/requests.html?created=" + req.id;
      } catch (err) {
        toastError(err);
        if (err.code === "empty_cart") {
          await loadCart();
        }
      }
    });
  }
});

document.getElementById("kind-filter").addEventListener("click", (event) => {
  const chip = event.target.closest(".chip");
  if (!chip) {
    return;
  }
  state.kind = chip.dataset.kind;
  document.querySelectorAll("#kind-filter .chip").forEach((c) => c.classList.toggle("active", c === chip));
  render();
});

async function moveTo(start) {
  state.start = start < today ? today : start;
  startInput.value = state.start;
  document.getElementById("prev-week").disabled = state.start <= today;
  await loadSchedule();
  render();
}

startInput.min = today;
startInput.value = today;
startInput.addEventListener("change", () => startInput.value && moveTo(startInput.value));
document.getElementById("prev-week").addEventListener("click", () => moveTo(addDays(state.start, -DAYS)));
document.getElementById("next-week").addEventListener("click", () => moveTo(addDays(state.start, DAYS)));
document.getElementById("prev-week").disabled = true;
if (!user) {
  document.querySelector("#schedule-panel h2 .row").hidden = true;
}

setInterval(() => {
  if (!state.cart.size) {
    return;
  }
  state.expiresIn -= 1;
  if (state.expiresIn <= 0) {
    toast("Корзина истекла", "info");
    loadCart().then(render);
    return;
  }
  const ttl = document.getElementById("ttl");
  const bar = document.getElementById("ttl-bar");
  if (ttl) {
    ttl.textContent = formatDuration(state.expiresIn);
  }
  if (bar && state.maxTTL) {
    bar.style.width = Math.min(100, (state.expiresIn / state.maxTTL) * 100) + "%";
  }
}, 1000);

await Promise.all([loadServices(), loadSchedule(), loadCart()]);
render();
