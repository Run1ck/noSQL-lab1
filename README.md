<div align="center">

# UniBooking

**Бронирование аудиторий, лабораторий и оборудования**

Сервис на Go: пользователь собирает корзину из свободных дней, отправляет заявку, администратор её одобряет или отклоняет.
Постоянные данные лежат в PostgreSQL, временные (корзина, кэш, лимиты) — в Redis.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker_Compose-2496ED?logo=docker&logoColor=white)

<img src="docs/screenshots/catalog.webp" alt="Каталог и корзина" width="900">

</div>

## Возможности

- **Каталог** аудиторий, лабораторий и оборудования с фильтром по типу.
- **Расписание** на неделю: сетка «услуги × дни», занятые дни видно сразу.
- **Корзина** живёт в Redis с TTL: брошенная корзина исчезает сама, обратный отсчёт виден в интерфейсе.
- **Заявки**: оформление из корзины, история со статусами и комментарием администратора, отмена, пока заявка не обработана.
- **Админка**: обработка заявок и управление каталогом (создать, переименовать, скрыть).
- **Защита**: JWT-авторизация, rate limit на весь API по IP и бан за частые заявки — оба на Lua-скриптах в Redis.
- **Кэш** каталога и расписания поверх Postgres с точечной инвалидацией. Отключается одной переменной, чтобы сравнивать замеры.
- **UI** без фреймворков и сборки: HTML, CSS и ES-модули, встроенные в бинарник через `embed`. Тема — glassmorphism, адаптив до мобильных.

## Стек

| | |
|---|---|
| Язык | Go 1.26, только стандартный `net/http` (маршрутизация через `ServeMux` с методами и шаблонами путей) |
| PostgreSQL 18 | [pgx/v5](https://github.com/jackc/pgx) + пул `pgxpool`, типобезопасные запросы через [sqlc](https://sqlc.dev) |
| Redis 7 | [go-redis/v9](https://github.com/redis/go-redis), Lua-скрипты для атомарных счётчиков, AOF + RDB |
| Авторизация | [golang-jwt/v5](https://github.com/golang-jwt/jwt) (HS256), пароли — bcrypt |
| Тесты | `testing`, [testify](https://github.com/stretchr/testify), моки [mockery](https://vektra.github.io/mockery/), [miniredis](https://github.com/alicebob/miniredis) |
| Инфраструктура | Docker Compose |

## Архитектура

Чистая архитектура: зависимости направлены внутрь, один use case — один файл.
Use case'ы знают только интерфейсы `Postgres` и `Redis`, поэтому кэш подключается
декоратором, и логика не знает, есть он или нет.

```mermaid
flowchart LR
    UI["web/ — UI<br/>(embed)"] --> MW
    subgraph HTTP["controller/http"]
        MW["middleware<br/>rate limit · JWT · admin"] --> H["v1 handlers"]
    end
    H --> UC["usecase<br/>один файл — один сценарий"]
    UC -->|usecase.Postgres| CACHE["adapter/repository<br/>кэш-декоратор"]
    CACHE --> PGA["adapter/postgres<br/>sqlc"]
    UC -->|usecase.Redis| RA["adapter/redis<br/>корзина"]
    UC --> BAN["pkg/ratelimit<br/>бан"]
    PGA --> PG[(PostgreSQL)]
    CACHE --> R[(Redis)]
    RA --> R
    BAN --> R
    MW --> R
```

### Путь заявки

```mermaid
sequenceDiagram
    actor U as Пользователь
    participant API
    participant R as Redis
    participant PG as Postgres
    actor A as Админ

    U->>API: POST /api/cart/items
    API->>R: SADD cart:{user} + EXPIRE
    U->>API: POST /api/requests
    API->>R: бан-счётчик (Lua)
    API->>PG: INSERT заявки и позиций (транзакция)
    API->>R: DEL cart:{user}
    A->>API: POST /api/admin/requests/{id}/approve
    API->>PG: UPDATE … WHERE status = 'new' + INSERT bookings
    API->>R: DEL cache:schedule:{день}
```

Гонки решает база, а не код:
- два админа жмут «Одобрить» одновременно — условный `UPDATE … WHERE status = 'new'` пропустит только одного, второй получит `409 already_processed`;
- две заявки на один день — первичный ключ `bookings (service_id, date)` не даст одобрить вторую, ответ `409 slot_booked`.

Корзина и заявки в статусе `new` день не занимают — занимает только одобрение.

## Быстрый старт

Нужны Go 1.26+ и Docker.

```bash
cp deploy/.env.example deploy/.env   # поменяйте JWT_SECRET и ADMIN_PASSWORD
make up-dev                          # PostgreSQL + Redis + эмулятор Firestore + Adminer
make run                             # приложение на http://localhost:8080
```

Для разработки — `deploy/devcompose.yaml`: подключает продовый `compose.yaml` и
добавляет эмулятор Firestore на `127.0.0.1:8081` и Adminer на http://localhost:9000
(сервер `postgres`, логин и пароль из `deploy/.env`). Продовый `deploy/compose.yaml`
поднимает только PostgreSQL и Redis: Firestore в проде — облачный.

Схема БД применяется автоматически из `migration/postgres/init.sql` при первом старте контейнера.
Администратор создаётся при запуске приложения из `ADMIN_LOGIN` / `ADMIN_PASSWORD`,
каталог услуг заполняется в админке.

| Команда | Что делает |
|---|---|
| `make up` | поднять продовый набор: PostgreSQL и Redis |
| `make up-dev` | поднять набор для разработки: плюс эмулятор Firestore и Adminer |
| `make down` | остановить всё, что поднято любой из команд выше |
| `make run` | запустить приложение |
| `make test` | все тесты с `-race` |
| `make keys` | ключи Redis с типами и оставшимся TTL |
| `make flush` | очистить Redis |
| `make generate` | перегенерировать моки (`make mockery-install` — один раз) |

Код запросов генерируется sqlc из `internal/adapter/postgres/queries/*.sql`:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

## Конфигурация

Переменные окружения или `deploy/.env`.

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `PORT` | `:8080` | адрес HTTP-сервера |
| `POSTGRES_DSN` | `postgres://booking:booking@127.0.0.1:5432/booking` | подключение к PostgreSQL |
| `REDIS_HOST`, `REDIS_PORT` | `127.0.0.1`, `6379` | подключение к Redis |
| `JWT_SECRET` | — | ключ подписи токенов, **обязателен** |
| `JWT_TTL` | `24h` | срок жизни токена |
| `ADMIN_LOGIN`, `ADMIN_PASSWORD` | `admin`, — | администратор, создаётся при старте |
| `CART_TTL` | `30m` | время жизни корзины, продлевается при каждом изменении |
| `CACHE_ENABLED` | `true` | кэш каталога и расписания |
| `CACHE_TTL` | `60s` | время жизни записей кэша |
| `RATE_LIMIT`, `RATE_WINDOW` | `60`, `1m` | запросов к API с одного IP за окно |
| `BAN_LIMIT`, `BAN_WINDOW`, `BAN_TTL` | `5`, `10m`, `15m` | больше `BAN_LIMIT` заявок за окно — бан на `BAN_TTL` |

## REST API

Все пути начинаются с `/api`, формат — JSON. Авторизация — `Authorization: Bearer <jwt>`.
Ошибки единообразны: `{"error": {"code": "slot_booked", "message": "…"}}`.

| Метод и путь | Доступ | Что делает |
|---|---|---|
| `POST /auth/register` | все | регистрация |
| `POST /auth/login` | все | вход, выдаёт JWT |
| `GET /auth/me` | пользователь | текущий пользователь |
| `GET /services` | все | активные услуги |
| `GET /schedule?from=&to=` | пользователь | занятые дни, до 31 дня |
| `GET /cart` | пользователь | корзина и остаток TTL |
| `POST /cart/items` | пользователь | добавить день услуги |
| `DELETE /cart/items/{serviceID}/{date}` | пользователь | убрать позицию |
| `DELETE /cart` | пользователь | очистить корзину |
| `POST /requests` | пользователь | оформить заявку из корзины |
| `GET /requests` | пользователь | мои заявки |
| `GET /requests/{id}` | пользователь | заявка (своя; админу — любая) |
| `POST /requests/{id}/cancel` | пользователь | отменить новую заявку |
| `GET /admin/requests?status=` | админ | заявки, фильтр по статусу |
| `POST /admin/requests/{id}/approve` | админ | одобрить, занять дни |
| `POST /admin/requests/{id}/reject` | админ | отклонить |
| `GET /admin/services` | админ | все услуги, включая скрытые |
| `PUT /admin/services/{id}` | админ | создать или изменить услугу |

Полный контракт с телами запросов, кодами ошибок и бизнес-правилами — в [docs/CONTRACTS.md](docs/CONTRACTS.md).

## Данные в Redis

| Ключ | Тип | TTL | Что хранит |
|---|---|---|---|
| `cart:{userID}` | SET `serviceID\|YYYY-MM-DD` | `CART_TTL` | корзина |
| `cache:services` | STRING (JSON) | `CACHE_TTL` | каталог, сбрасывается при изменении услуги |
| `cache:schedule:{YYYY-MM-DD}` | STRING (JSON) | `CACHE_TTL` | брони дня, сбрасываются при одобрении заявки на этот день |
| `rl:api:{ip}` | счётчик | `RATE_WINDOW` | лимит API |
| `rl:req:{userID}` | счётчик | `BAN_WINDOW` | заявки пользователя за окно |
| `ban:{userID}` | STRING | `BAN_TTL` | активный бан |

Redis работает с `maxmemory 64mb` и `volatile-lru`, а данные переживают рестарт благодаря AOF + RDB.
Кэш и лимитер работают по принципу fail-open: если Redis недоступен, запросы идут напрямую в Postgres
и не блокируются.

## Структура проекта

```
cmd/app/                 точка входа: config → app.Run
config/                  конфигурация из окружения
migration/postgres/      схема БД
internal/
  app/                   сборка зависимостей, HTTP-сервер, graceful shutdown
  controller/http/       маршруты и хендлеры (v1/ — файл на хендлер)
  usecase/               сценарии, интерфейсы Postgres и Redis, моки
  dto/                   входы и выходы use case'ов, формат JSON
  domain/                сущности и доменные ошибки
  auth/                  принципал и JWT
  adapter/
    postgres/            репозитории на pgx, запросы и код sqlc
    redis/               корзина
    repository/          кэш-декоратор поверх postgres
pkg/
  httpx/                 JSON, ошибки, middleware авторизации и лимита
  ratelimit/             лимитер и бан на Lua
  postgres/, redis/      клиенты
web/                     UI: статика, встроенная через embed
deploy/                  docker compose, redis.conf, .env
```

## Скриншоты

<table>
  <tr>
    <td><img src="docs/screenshots/schedule.webp" alt="Расписание"></td>
    <td><img src="docs/screenshots/requests.webp" alt="Мои заявки"></td>
  </tr>
  <tr>
    <td align="center">Расписание: занятые дни и выбранные в корзину</td>
    <td align="center">Мои заявки со статусами и ответом администратора</td>
  </tr>
  <tr>
    <td><img src="docs/screenshots/admin-requests.webp" alt="Админка: заявки"></td>
    <td><img src="docs/screenshots/admin-services.webp" alt="Админка: услуги"></td>
  </tr>
  <tr>
    <td align="center">Админка: обработка заявок</td>
    <td align="center">Админка: каталог услуг</td>
  </tr>
  <tr>
    <td><img src="docs/screenshots/login.webp" alt="Вход"></td>
    <td align="center"><img src="docs/screenshots/mobile.webp" alt="Мобильная версия" width="260"></td>
  </tr>
  <tr>
    <td align="center">Вход и регистрация</td>
    <td align="center">Мобильная версия</td>
  </tr>
</table>

## Авторы

Лабораторная работа по NoSQL, выполнена вдвоём:

- **Loskutov P.A.** — PostgreSQL, JWT-авторизация, заявки и админка, UI, сборка приложения;
- **[Run1ck](https://github.com/Run1ck)** — Redis: корзина, rate limit и бан, каталог и расписание.
