import { api, errorText, getUser, mountShell, setSession } from "./common.js";

mountShell("");

const params = new URLSearchParams(location.search);
const next = params.get("next")?.startsWith("/") ? params.get("next") : "/";

if (getUser()) {
  location.replace(next);
}

const form = document.getElementById("auth-form");
const submit = document.getElementById("submit");
const errorBox = document.getElementById("form-error");
const password = form.elements.password;
let mode = "login";

document.querySelectorAll(".tabs button").forEach((tab) => {
  tab.addEventListener("click", () => {
    mode = tab.dataset.mode;
    document.querySelectorAll(".tabs button").forEach((t) => t.classList.toggle("active", t === tab));
    submit.textContent = mode === "login" ? "Войти" : "Создать аккаунт";
    password.autocomplete = mode === "login" ? "current-password" : "new-password";
    errorBox.textContent = "";
  });
});

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  errorBox.textContent = "";

  const credentials = {
    login: form.elements.login.value.trim(),
    password: password.value,
  };
  if (!credentials.login || !credentials.password) {
    errorBox.textContent = "Заполните логин и пароль";
    return;
  }

  submit.disabled = true;
  try {
    if (mode === "register") {
      await api("POST", "/auth/register", credentials);
    }
    setSession(await api("POST", "/auth/login", credentials));
    location.replace(next);
  } catch (err) {
    errorBox.textContent = errorText(err);
  } finally {
    submit.disabled = false;
  }
});
