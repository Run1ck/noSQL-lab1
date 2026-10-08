const TOKEN_KEY = "booking.token";
const USER_KEY = "booking.user";
const EXPIRES_KEY = "booking.expires";

const ERRORS = {
  bad_request: "Некорректный запрос",
  unauthorized: "Нужно войти в систему",
  invalid_token: "Сессия истекла, войдите заново",
  forbidden: "Недостаточно прав",
  invalid_credentials: "Неверный логин или пароль",
  invalid_password: "Пароль должен быть от 8 до 72 символов",
  invalid_role: "Неизвестная роль",
  login_taken: "Этот логин уже занят",
  user_not_found: "Пользователь не найден",
  invalid_service: "Укажите ID и название услуги",
  invalid_kind: "Неизвестный тип услуги",
  invalid_date: "Некорректная дата",
  service_not_found: "Услуга не найдена",
  invalid_id: "Некорректный идентификатор",
  item_not_found: "Позиции нет в корзине",
  past_date: "Нельзя бронировать прошедший день",
  service_unavailable: "Услуга сейчас недоступна",
  slot_booked: "Этот день уже занят",
  empty_cart: "Корзина пуста",
  invalid_status: "Неизвестный статус",
  request_not_found: "Заявка не найдена",
  already_processed: "Заявка уже обработана",
  rate_limited: "Слишком много запросов",
  banned: "Слишком много заявок подряд",
  internal: "Ошибка сервера, попробуйте позже",
};

export const KINDS = {
  room: "Аудитория",
  lab: "Лаборатория",
  equipment: "Оборудование",
};

export const STATUSES = {
  new: "Новая",
  approved: "Одобрена",
  rejected: "Отклонена",
  cancelled: "Отменена",
};

export class ApiError extends Error {
  constructor(status, code, message, retryAfter) {
    super(message);
    this.status = status;
    this.code = code;
    this.retryAfter = retryAfter;
  }
}

function read(key) {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key, value) {
  try {
    if (value === null) {
      localStorage.removeItem(key);
    } else {
      localStorage.setItem(key, value);
    }
  } catch {}
}

export function getToken() {
  const token = read(TOKEN_KEY);
  const expires = Date.parse(read(EXPIRES_KEY) || "");
  if (!token || (expires && expires <= Date.now())) {
    return null;
  }
  return token;
}

export function getUser() {
  if (!getToken()) {
    return null;
  }
  try {
    return JSON.parse(read(USER_KEY) || "null");
  } catch {
    return null;
  }
}

export function setSession({ token, expires_at, user }) {
  write(TOKEN_KEY, token);
  write(EXPIRES_KEY, expires_at);
  write(USER_KEY, JSON.stringify(user));
}

export function clearSession() {
  write(TOKEN_KEY, null);
  write(EXPIRES_KEY, null);
  write(USER_KEY, null);
}

export function goLogin() {
  const next = location.pathname + location.search;
  location.href = "/login.html?next=" + encodeURIComponent(next);
}

export async function api(method, path, body) {
  const headers = { Accept: "application/json" };
  const token = getToken();
  if (token) {
    headers.Authorization = "Bearer " + token;
  }
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  const resp = await fetch("/api" + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  let data = null;
  if ((resp.headers.get("Content-Type") || "").includes("application/json")) {
    data = await resp.json();
  }

  if (!resp.ok) {
    const code = data?.error?.code || "internal";
    const retryAfter = Number(resp.headers.get("Retry-After")) || 0;
    if (resp.status === 401 && token && code === "invalid_token") {
      clearSession();
      goLogin();
    }
    throw new ApiError(resp.status, code, data?.error?.message || resp.statusText, retryAfter);
  }

  return data;
}

export function errorText(err) {
  if (!(err instanceof ApiError)) {
    return "Нет связи с сервером";
  }
  let text = ERRORS[err.code] || err.message;
  if (err.retryAfter) {
    text += ". Повторите через " + formatDuration(err.retryAfter);
  }
  return text;
}

export function formatDuration(seconds) {
  const s = Math.max(0, Math.round(seconds));
  const m = Math.floor(s / 60);
  if (m >= 60) {
    return Math.floor(m / 60) + " ч " + (m % 60) + " мин";
  }
  if (m > 0) {
    return m + " мин " + (s % 60) + " с";
  }
  return s + " с";
}

export function esc(value) {
  return String(value ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
}

function toastBox() {
  let box = document.querySelector(".toasts");
  if (!box) {
    box = document.createElement("div");
    box.className = "toasts";
    box.setAttribute("role", "status");
    box.setAttribute("aria-live", "polite");
    document.body.append(box);
  }
  return box;
}

export function toast(message, kind = "info") {
  const el = document.createElement("div");
  el.className = "toast glass " + kind;
  el.textContent = message;
  toastBox().append(el);
  setTimeout(() => el.classList.add("hide"), 3600);
  setTimeout(() => el.remove(), 4000);
}

export function toastError(err) {
  toast(errorText(err), "bad");
}

export function todayISO() {
  const d = new Date();
  return toISO(new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate())));
}

export function toISO(date) {
  return date.toISOString().slice(0, 10);
}

export function addDays(iso, n) {
  const d = new Date(iso + "T00:00:00Z");
  d.setUTCDate(d.getUTCDate() + n);
  return toISO(d);
}

const dayFmt = new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short", timeZone: "UTC" });
const dowFmt = new Intl.DateTimeFormat("ru-RU", { weekday: "short", timeZone: "UTC" });
const longFmt = new Intl.DateTimeFormat("ru-RU", { weekday: "short", day: "numeric", month: "long", timeZone: "UTC" });
const dateTimeFmt = new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });

export function formatDay(iso) {
  return dayFmt.format(new Date(iso + "T00:00:00Z"));
}

export function formatDow(iso) {
  return dowFmt.format(new Date(iso + "T00:00:00Z"));
}

export function formatLongDay(iso) {
  return longFmt.format(new Date(iso + "T00:00:00Z"));
}

export function formatDateTime(iso) {
  return iso ? dateTimeFmt.format(new Date(iso)) : "";
}

export function statusBadge(status) {
  return `<span class="badge ${esc(status)}">${esc(STATUSES[status] || status)}</span>`;
}

export function mountShell(active) {
  const blobs = document.createElement("div");
  blobs.className = "blobs";
  blobs.innerHTML = '<div class="blob b1"></div><div class="blob b2"></div><div class="blob b3"></div>';
  document.body.prepend(blobs);

  const nav = document.getElementById("nav");
  if (!nav) {
    return;
  }

  const user = getUser();
  const links = [["catalog", "/", "Каталог"]];
  if (user) {
    links.push(["requests", "/requests.html", "Мои заявки"]);
  }
  if (user?.role === "admin") {
    links.push(["admin", "/admin.html", "Админка"]);
  }

  const linksHTML = links
    .map(([key, href, title]) => `<a href="${href}" class="${key === active ? "active" : ""}">${title}</a>`)
    .join("");

  const userHTML = user
    ? `<span class="avatar" aria-hidden="true">${esc(user.login.slice(0, 1).toUpperCase())}</span>
       <span>${esc(user.login)}${user.role === "admin" ? ' <span class="faint">· админ</span>' : ""}</span>
       <button class="btn sm ghost" id="logout">Выйти</button>`
    : `<a class="btn sm primary" href="/login.html">Войти</a>`;

  nav.className = "nav glass container";
  nav.innerHTML = `
    <a class="brand" href="/"><span class="brand-mark"></span><span>Booking</span></a>
    <nav class="nav-links">${linksHTML}</nav>
    <div class="nav-user">${userHTML}</div>`;

  nav.querySelector("#logout")?.addEventListener("click", () => {
    clearSession();
    location.href = "/";
  });
}

export function requireUser({ admin = false } = {}) {
  const user = getUser();
  if (!user) {
    goLogin();
    return null;
  }
  if (admin && user.role !== "admin") {
    location.href = "/";
    return null;
  }
  return user;
}
