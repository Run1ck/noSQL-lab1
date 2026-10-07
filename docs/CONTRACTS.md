# Распределение работы и контракты

## Принцип деления

Делим по хранилищу:

- **А — Redis.** Всё, что живёт в Redis: корзина, кэши, rate limiter, бан. Плюс
  read-side поверх кэша (каталог, расписание), исследование, нагрузочный тест
  и выводы — то есть ядро лабораторной.
- **Б — Postgres.** Всё, что живёт в Postgres: пользователи, заявки, брони.
  Плюс JWT-авторизация, обработка заявок админом, UI и сборка приложения.

Каждый владеет своими пакетами целиком (хранилище → логика → HTTP). В чужой
код не ходим — только через интерфейсы из раздела «Точки стыка».

## Кто что делает

### А — Redis

| Задача | Где |
|---|---|
| Подключение к Redis | `pkg/redis` |
| Корзина в Redis: SET + TTL | `internal/adapter/redis/` |
| Кэш услуг и броней поверх Postgres, инвалидация расписания на Approve | `internal/adapter/repository/` |
| `ratelimit.Limiter` на Redis (Lua): лимит API и бан | `pkg/ratelimit/` |
| Middleware лимита + `ClientIP` | `pkg/httpx/ratelimit_mw.go` |
| Use case'ы: каталог, расписание, корзина | `internal/usecase/{get_services,get_schedule,get_cart,add_cart_item,remove_cart_item,clear_cart}.go`, DTO — `internal/dto/` |
| HTTP: каталог, расписание, корзина | `internal/controller/http/v1/` |
| Сохранение и восстановление (AOF + RDB), проверка рестартом | `deploy/redis.conf` |
| Исследование «потеря временных ключей» + выводы о применимости | `research/` |
| Нагрузочный тестер на C++ | `tools/loadtest/` |

Исследование, минимум:
- естественное истечение TTL: корзина пропала → `POST /api/requests` отвечает `empty_cart`;
- вытеснение при `maxmemory 64mb` / `volatile-lru`: забить память ключами с TTL
  и посмотреть, что пропадает. Кэш — безопасно, просто промах и запрос в БД.
  Корзина — пользователь её теряет. Счётчик лимита и бан — сбрасываются раньше
  срока, то есть защита ослабевает;
- рестарт с AOF и без: что восстановилось, с каким остатком TTL, что истекло за простой.

### Б — Postgres, авторизация, заявки, UI

| Задача | Где |
|---|---|
| Postgres в compose, схема | `deploy/compose.yaml` (сервис `postgres`), `migration/postgres/init.sql` |
| Пул Postgres | `pkg/postgres` |
| Методы `usecase.Postgres`: пользователи, услуги, заявки, брони | `internal/adapter/postgres/` (запросы — `queries/`, код sqlc — `sqlc/`, конфиг — `sqlc.yaml` в корне) |
| `auth.Tokens` на JWT | `internal/auth/jwt.go` |
| Middleware `Authenticate`, `RequireAdmin` | `pkg/httpx/auth_mw.go` |
| Use case'ы: регистрация и вход, заявки пользователя, админка | `internal/usecase/`, DTO — `internal/dto/` |
| HTTP: регистрация и вход, заявки пользователя, админка | `internal/controller/http/v1/` |
| Создание админа при старте (`ADMIN_LOGIN` / `ADMIN_PASSWORD`) | use case, вызов из `internal/app` |
| UI: статический HTML + JS через `embed` | `web/` |
| Сборка приложения | `cmd/app/main.go` → `internal/app/app.go` |

Сервис `postgres` в compose: контейнер `booking-postgres`, пользователь, пароль
и база — `booking`, порт 5432, `init.sql` монтируется в `/docker-entrypoint-initdb.d/`.
С этими значениями совпадает DSN по умолчанию в конфиге.

Если Б не успевает с UI, А забирает страницы каталога и корзины (`web/catalog.*`).
Б заранее выносит общий fetch и хранение токена в `web/common.js`.

## Общая зона

Эти файлы меняются только отдельным PR `contract: …` с апрувом второго:

- `internal/domain/**`
- `internal/auth/auth.go`
- `pkg/ratelimit/ratelimit.go`
- `pkg/httpx/{json,errors}.go` (`module.go` больше не используется — маршруты в `router.go`)
- `internal/usecase/usecase.go` — интерфейсы `Postgres`, `Redis`, `UseCase`, `New`: каждый дописывает свои методы
- `internal/controller/http/router.go`, `internal/controller/http/v1/v1.go` — каждый дописывает свои маршруты
- `.mockery.yml`, `internal/usecase/mocks/` (генерируется `make generate`)
- `config/config.go`, `deploy/.env.example`
- этот файл

Остальное:
- `go.mod` / `go.sum`: конфликт решается `go mod tidy`;
- `cmd/app/main.go` принадлежит Б. А присылает свои строки сборки в PR, в блок `// Redis`;
- `deploy/compose.yaml`: А правит сервис `redis`, Б — сервис `postgres`.

## Слои

Стиль — как в эталоне чистой архитектуры (проект `my-app`): правило зависимостей
направлено внутрь, один use case — один файл.

```
cmd/app/main.go             config → app.Run
config                      конфиг из env
migration/postgres          схема БД
internal/app                сборка: адаптеры → usecase.New → controller/http.Router
internal/controller/http    router.go (маршруты) + v1/: Handlers, файл на хендлер
internal/usecase            usecase.go (интерфейсы Postgres, Redis; UseCase; New) + файл на сценарий
internal/dto                Input/Output use case'ов; json-теги — формат ответов REST API
internal/domain             сущности и доменные ошибки (общая зона)
internal/adapter            redis (А), postgres (Б), repository — кэш Redis поверх postgres (А)
pkg/redis, pkg/postgres     клиенты
pkg/httpx                   JSON, ошибки (errorMap), middleware лимита и авторизации
pkg/ratelimit               лимитер на Redis (Lua): интерфейс Limiter, окно, бан
```

- Use case: `func (u *UseCase) X(ctx, dto.XInput) (dto.XOutput, error)`. Разбор
  строк (даты, ID) — в use case'е, ошибки — доменные, остальное оборачивается:
  `fmt.Errorf("u.redis.GetCart: %w", err)`.
- Хендлер: собрать `dto.XInput` из запроса (JSON — `httpx.DecodeJSON`, путь,
  query, пользователь — `auth.FromContext`), вызвать use case, ответить
  `httpx.WriteJSON` или `httpx.WriteError` (статус и код — из `errorMap`).
- Тесты use case'ов — пакет `usecase_test` и моки mockery (`internal/usecase/mocks`).

## Точки стыка

| Контракт | Реализует | Использует |
|---|---|---|
| `usecase.Redis`: корзина | А (`adapter/redis`) | use case'ы корзины (А); оформление заявки (Б: `GetCart`, `DeleteCart`) |
| `usecase.Postgres` | Б (`adapter/postgres`); А оборачивает кэшем (`adapter/repository`) | все use case'ы |
| `ratelimit.Limiter` | А (Redis) | А: middleware; Б: бан при оформлении заявки |
| `auth.Tokens`, middleware авторизации | Б | А: `auth.FromContext` в хендлерах корзины |
| `controller/http.Router`, `httpx.Middlewares` | оба — свои маршруты | `app` |

Методы `usecase.Postgres`, которые нужны А (реализует Б):

```go
GetServices(ctx) ([]*service.Service, error)          // все, включая неактивные
GetService(ctx, id string) (*service.Service, error)  // нет — service.ErrNotFound
GetBookings(ctx, date service.Date) ([]booking.Booking, error)
```

`adapter/repository` реализует тот же `usecase.Postgres`: читает из Redis-кэша и
ходит в Postgres Б на промахе, поэтому use case'ы не знают, есть ли кэш. При
`CACHE_ENABLED=false` `app` передаёт `adapter/postgres` напрямую — это нужно для
замеров.

Пока чужая часть не готова, use case'ы тестируются на моках интерфейсов, а
хендлеры — с заглушкой, которая кладёт фиксированный `Principal` через
`auth.WithPrincipal`.

## Сборка в main

Имена конструкторов — договорённость, по ним Б собирает приложение в `internal/app`:

```go
// Redis (А)
client, err := redis.New(ctx, cfg.RedisAddr) // pkg/redis, *redis.Client
apiLimit := ratelimit.NewRedis(client, "rl:api", cfg.RateLimit, cfg.RateWindow)
banLimit := ratelimit.NewBan(client, cfg.BanLimit, cfg.BanWindow, cfg.BanTTL)

// Postgres (Б)
pool, err := postgres.New(ctx, cfg.PostgresDSN) // pkg/postgres
var pg usecase.Postgres = adapterpostgres.New(pool)
tokens := auth.NewJWT(cfg.JWTSecret, cfg.JWTTTL) // auth.Tokens

// Кэш (А) поверх Postgres Б
if cfg.CacheEnabled {
	pg = repository.New(client, pg, cfg.CacheTTL) // Approve → DEL cache:schedule:{date}
}

// UseCase
uc := usecase.New(pg, adapterredis.New(client, cfg.CartTTL) /* Б: tokens, banLimit */)

// HTTP
authn := httpx.Authenticate(tokens)
mw := httpx.Middlewares{
	Auth:  authn,
	Admin: func(h http.Handler) http.Handler { return authn(httpx.RequireAdmin(h)) },
}
mux := http.NewServeMux()
controllerhttp.Router(mux, uc, mw)
mux.Handle("/", web.Handler()) // Б

// Лимит только на /api/*, статику UI не считаем.
handler := httpx.RateLimit(apiLimit, httpx.ClientIP)(mux) // А
```

## REST API

Префикс `/api`, JSON в UTF-8. Авторизация — заголовок `Authorization: Bearer <jwt>`,
UI хранит токен в `localStorage`. Даты — `YYYY-MM-DD` (так сериализуется
`service.Date`), время — RFC 3339.

**Доступ:** «все» — без токена; Auth — с токеном; Admin — с токеном администратора.

### Объекты

```jsonc
// User
{"id": "8f1c…", "login": "ivanov", "role": "user"}

// Service
{"id": "room-101", "name": "Аудитория 101", "kind": "room", "active": true}

// Cart: позиции отсортированы по дате, затем по service_id;
// expires_in — секунд до истечения корзины, 0 если корзины нет
{"items": [{"service_id": "room-101", "date": "2026-10-05"}], "expires_in": 1795}

// Request
{
  "id": 42,
  "user_id": "8f1c…",
  "items": [{"service_id": "room-101", "date": "2026-10-05"}],
  "status": "new",              // new | approved | rejected | cancelled
  "comment": "",
  "created_at": "2026-10-03T12:00:00Z",
  "processed_at": null,
  "processed_by": null
}
```

### Авторизация — Б

| Метод и путь | Доступ | Тело | Ответ | Ошибки |
|---|---|---|---|---|
| `POST /api/auth/register` | все | `{"login", "password"}` | `201 User` | 400 `invalid_password`, 409 `login_taken` |
| `POST /api/auth/login` | все | `{"login", "password"}` | `200 {"token", "expires_at", "user": User}` | 401 `invalid_credentials` |
| `GET /api/auth/me` | Auth | — | `200 User` | 401 |

### Каталог и расписание — А

| Метод и путь | Доступ | Ответ | Ошибки |
|---|---|---|---|
| `GET /api/services` | все | `200 [Service]`, только активные | — |
| `GET /api/schedule?from=…&to=…` | Auth | `200 {"days": [{"date": "2026-10-05", "booked": ["room-101"]}]}` | 400 `invalid_date`, 400 `bad_request` (диапазон больше 31 дня или `from > to`) |

В `days` есть каждый день диапазона включительно, у свободного — `"booked": []`.

### Корзина — А, всё под Auth

| Метод и путь | Тело | Ответ | Ошибки |
|---|---|---|---|
| `GET /api/cart` | — | `200 Cart` | — |
| `POST /api/cart/items` | `{"service_id", "date"}` | `200 Cart` | 400 `invalid_date`, 404 `service_not_found`, 422 `service_unavailable`, 422 `past_date`, 409 `slot_booked` |
| `DELETE /api/cart/items/{serviceID}/{date}` | — | `200 Cart` | 404 `item_not_found` |
| `DELETE /api/cart` | — | `204` | — |

### Заявки пользователя — Б, всё под Auth

| Метод и путь | Тело | Ответ | Ошибки |
|---|---|---|---|
| `POST /api/requests` | — | `201 Request` из корзины, корзина удаляется | 422 `empty_cart`, 429 `banned` |
| `GET /api/requests` | — | `200 [Request]` свои, новые сверху | — |
| `GET /api/requests/{id}` | — | `200 Request` | 403 `forbidden` (чужая; админу можно), 404 `request_not_found` |
| `POST /api/requests/{id}/cancel` | — | `200 Request` | 403 `forbidden`, 404, 409 `already_processed` |

### Админка — Б, всё под Admin

| Метод и путь | Тело | Ответ | Ошибки |
|---|---|---|---|
| `GET /api/admin/requests?status=new` | — | `200 [Request]`; без `status` — все | 400 `invalid_status` |
| `POST /api/admin/requests/{id}/approve` | `{"comment"}` | `200 Request` | 404, 409 `already_processed`, 409 `slot_booked` |
| `POST /api/admin/requests/{id}/reject` | `{"comment"}` | `200 Request` | 404, 409 `already_processed` |
| `GET /api/admin/services` | — | `200 [Service]`, включая неактивные | — |
| `PUT /api/admin/services/{id}` | `{"name", "kind", "active"}` | `200 Service`, создаёт или обновляет | 400 `invalid_service`, 400 `invalid_kind` |

### Общее для всех `/api/*`

- Ошибка: `{"error": {"code": "…", "message": "…"}}`. Коды и статусы — в
  `pkg/httpx/errors.go`, ответ пишется только через `httpx.WriteError`.
- 401 `unauthorized` — нет токена; 401 `invalid_token` — токен битый или истёк;
  403 `forbidden` — нужен администратор.
- Любой запрос может получить 429 `rate_limited` с `Retry-After` в секундах.
  Пропущенные запросы несут `X-RateLimit-Limit` и `X-RateLimit-Remaining`.
- Битый JSON, лишние поля, нечисловой `{id}` — 400 `bad_request`.

## Ключи Redis — А

| Ключ | Тип | TTL | Что хранит |
|---|---|---|---|
| `cart:{userID}` | SET из `serviceID\|YYYY-MM-DD` | `CART_TTL`, продлевается при каждом изменении | корзина |
| `cache:services` | STRING, JSON | `CACHE_TTL` | все услуги; DEL при сохранении услуги |
| `cache:schedule:{YYYY-MM-DD}` | STRING, JSON | `CACHE_TTL` | брони дня; DEL при `Approve` заявки с этим днём |
| `rl:api:{ip}` | счётчик | `RATE_WINDOW` | лимит API |
| `rl:req:{userID}` | счётчик | `BAN_WINDOW` | заявки пользователя за окно |
| `ban:{userID}` | STRING | `BAN_TTL` | активный бан |

Почему так:
- корзина — SET: пара (услуга, день) уникальна сама собой, `SADD`/`SREM` за O(1),
  вся корзина истекает одним TTL;
- кэши — JSON-строка: читаются целиком одним `GET`;
- счётчики — `INCR` + `PEXPIRE` в одном Lua-скрипте, это и есть атомарность.

У всех ключей есть TTL, значит при `maxmemory` все они кандидаты на вытеснение
под `volatile-lru`. Это и есть предмет исследования.

## Бизнес-правила

- **Добавить в корзину** (А, `usecase.AddCartItem`) можно только существующую услугу (`service_not_found`),
  если она активна (`service_unavailable`), на день не раньше сегодняшнего
  (`past_date`), не занятый одобренной бронью (`slot_booked`). Повторное
  добавление той же пары — не ошибка.
- **Расписание** (А, `usecase.GetSchedule`): `from > to` или больше 31 дня —
  `service.ErrInvalidRange` → 400 `bad_request`.
- Корзина и заявки в статусе `new` день не занимают. Занимает только одобрение
  (см. комментарий в `init.sql`).
- **Оформление** (Б), по шагам:
  1. `u.redis.GetCart`; если корзина пуста — `empty_cart`.
  2. `banLimit.Allow(ctx, userID.String())`; при отказе вернуть
     `&ratelimit.LimitedError{Banned: true, RetryAfter: d.RetryAfter}`.
  3. `request.NewFromCart` → `requests.Create`.
  4. `u.redis.DeleteCart`. Ошибку здесь только пишем в лог — корзина всё равно истечёт по TTL.

  Если `Allow` вернул ошибку (Redis недоступен), пишем в лог и пропускаем.
- **Одобрение и отклонение** (Б): `u.postgres` — получить заявку с позициями →
  `r.Approve(adminID, comment)` → сохранить. Гонку двух админов отсекает БД (`already_processed`,
  `slot_booked`). Кэш расписания чистит декоратор А, Б ничего не вызывает.
- **Отмена**: только владелец и только из `new`.
- **Лимит API**: ключ — IP клиента, все `/api/*`. При недоступности Redis
  запрос пропускаем (fail-open).

## Порядок работы

1. Контракты (этот файл и код из «общей зоны») — готово.
2. Параллельно, без зависимостей друг от друга:
   - А: клиент Redis, корзина (`adapter/redis`), `Limiter` и бан, middleware
     лимита; use case'ы и хендлеры каталога, расписания и корзины на моках
     `usecase.Postgres` — сделано.
   - Б: Postgres в compose, `adapter/postgres`, JWT, авторизация; оформление
     заявки на моках `usecase.Redis` и `Limiter`.
3. Б собирает `internal/app`. Первый общий запуск — точка синхронизации.
4. Б делает UI; А — кэш (`adapter/repository`), нагрузочный тест, исследование, выводы.
5. Перед отчётом — общий прогон нагрузочного теста по живому приложению.
