# Часть А (Redis) — ход работы

Часть А из [`CONTRACTS.md`](CONTRACTS.md): всё, что живёт в Redis. Ветка `oleg-branch`.
Файлы общей зоны меняем только отдельным PR `contract: …` с апрувом Б.
Схема ключей Redis (тип, TTL, когда чистятся) — в [`CONTRACTS.md`, «Ключи Redis — А»](CONTRACTS.md#ключи-redis--а);
сюда не копируем, чтобы не разъехалось.

## Статус

| Шаг | Что | Статус |
|---|---|---|
| 0 | Redis в Docker (`make up`), конфиг проверен | готово |
| 1 | `redisstore.New` — подключение | готово |
| 2 | `redisstore.NewCarts` — `cart.Repository` | готово, не закоммичено |
| 3 | `ratelimit.NewRedis`, `ratelimit.NewBan` (Lua) | — |
| 4 | `httpx.RateLimit` + `ClientIP` | — |
| 5 | `catalogapi`, `cartapi`, `scheduleapi` на фейках | — |
| — | кэш-декораторы, исследование, C++ нагрузка, выводы | — |

## Окружение и проверки

- Redis: контейнер `booking-redis`, конфиг `deploy/redis.conf` (AOF `everysec` + RDB `save 60 100`,
  `maxmemory 64mb`, `volatile-lru`).
- Порт опубликован только на `127.0.0.1` (`deploy/compose.yaml`): пароля нет, из сети Redis не виден.
  Внутри контейнера `bind 0.0.0.0` остаётся — иначе не работает проброс порта Docker.
  После правок `compose.yaml` — `make up`: контейнер пересоздаётся, данные в volume `redis-data` целы.
- Проверить, что конфиг подхватился:
  `docker exec booking-redis redis-cli config get appendonly` → `yes`;
  `... config get maxmemory` → `67108864`; `... info persistence` → `aof_enabled:1`.
- Консоль: `docker exec -it booking-redis redis-cli`. Все ключи с типом и TTL: `make keys`.

## Тесты

- Юнит-тесты Redis-кода — на `miniredis` (`github.com/alicebob/miniredis/v2`): поднимается в
  процессе, умеет Lua и TTL; время двигаем `mr.FastForward(d)`, без `sleep`. `make test` работает
  без Docker.
- miniredis ≠ настоящий Redis: новый Lua/транзакции дополнительно гоняем на живом `booking-redis`
  (временной программой, которую потом удаляем). Живой Redis — и для исследования, и для нагрузки.
- API miniredis: `mr.Type(key)` возвращает одно значение; `mr.TTL(key)`, `mr.Exists(key)`, `mr.SAdd`.
- Отказ Redis имитируем `mr.SetError("ERR boom")` — все команды, включая `EVALSHA`, возвращают ошибку.
  Проверяем, что ошибка Redis не превращается в пустую корзину или `ErrItemNotFound` (404 вместо 500).
- Перед коммитом: `go mod tidy` и `go mod tidy -diff` (код 0 — `go.sum` в порядке).

## Решения по коду

- Соглашение: проверки вида `var _ Interface = (*T)(nil)` не пишем.
- Соглашение: у каждого Lua-скрипта в комментарии расписаны `KEYS[i]`, `ARGV[i]` и что он
  возвращает — сигнатуры у скрипта нет, связь с `Run(ctx, rdb, keys, args...)` только по позиции.

### `redisstore.New` (`internal/storage/redisstore/redis.go`)

- `PING` при создании: приложение падает на старте, а не на первом запросе. Неверный адрес —
  ошибка через ~2 с (go-redis сам делает ретраи и пишет их в лог).

### Корзина (`internal/storage/redisstore/cart.go`)

- Ключ `cart:{userID}`, SET из `serviceID|YYYY-MM-DD`. Разбор — по **последнему** `|`
  (в дате его нет, в ID услуги может быть).
- Пустой SET Redis удаляет сам → «корзины нет» и «корзина пуста» неразличимы; `Get` всегда
  отдаёт корзину (пустую, если ключа нет), `TTL` отдаёт 0 при `PTTL` -2/-1.
- `Save` — **перезапись** корзины: `DEL` + `SADD` + `PEXPIRE` в `MULTI/EXEC` (`TxPipelined`),
  чтобы ключ не остался без TTL. Пустая корзина → `DEL`.
- `Save` проверяет позиции (`ServiceID != ""`, `Date.IsValid()`): битая позиция не прочиталась бы
  обратно, и `Get` падал бы до конца TTL.
- `RemoveItem` — Lua (`removeItemScript`): `SREM`, и только если удалили — `PEXPIRE`. Промах → `ErrItemNotFound`,
  TTL не продлевается.
- Битая позиция в Redis → ошибка через `%v`, не `%w`: иначе `httpx.WriteError` вернёт
  400 `invalid_date` вместо 500.
- go-redis `PTTL`: для -1/-2 возвращает `time.Duration(-1/-2)` (наносекунды), не умноженные на ms.
- **Гонка при добавлении — принята.** Добавление в корзину есть только в `POST /api/cart/items`
  (`cartapi`): Get → `c.AddItem` → Save. Два параллельных добавления одного пользователя (две
  вкладки, быстрые клики в UI) могут потерять позицию: второй `Save` перезапишет корзину своим
  снимком. Лечится методом `cart.AddItem` (`SADD` + `PEXPIRE`) в контракте — решили не делать,
  для лабы не критично.

## Ревью 2026-10-04 — что учесть

Код корзины ревью прошёл без багов. Мелочи по нашему коду исправлены, остальное — ниже.

### Наш код — исправлено
Проверка позиций в `Save`, тесты ошибок Redis и битых данных, `removeItemScript`, doc-комментарии,
`go mod tidy`, порт Redis на `127.0.0.1` — см. разделы выше.

### На следующие шаги
- **`cartapi`:** `httpx.DecodeJSON` заворачивает ошибку через `%v`, поэтому плохая дата в JSON-поле
  типа `service.Date` даёт 400 `bad_request`, а не `invalid_date`. В DTO брать `Date string` и
  вызывать `service.ParseDate` после `DecodeJSON`.
- **Middleware лимита** оборачивает весь mux, включая `/` — сам пропускает пути не с `/api/`.
- **`ClientIP`:** только `net.SplitHostPort(r.RemoteAddr)`, без `X-Forwarded-For` (подделывается,
  лимит обходится).
- **Lua фиксированного окна:** `PEXPIRE`, если `n == 1` **или** `PTTL == -1` — иначе счётчик без TTL
  блокирует IP навсегда и не вытесняется под `volatile-lru`. `RetryAfter` — `PTTL` из того же скрипта.
- **Бан:** оба ключа (`rl:req:{uid}`, `ban:{uid}`) в `KEYS` одного скрипта; запрос, превысивший
  лимит, отклоняется, и в том же скрипте `SET ban … PX`.
- **TTL кэша (`CACHE_TTL=60s`, дефолт из контракта, не замерялся).** TTL нужен и при явной
  инвалидации (`DEL` на `Save`/`Approve`): страхует от гонки «промах → запись старого», от упавшего
  `DEL`, от правок в обход приложения; без TTL ключ не вытесняется под `volatile-lru`, а
  `cache:schedule:{date}` копились бы. 60 с: попаданий уже ~99,9% (1 запрос в БД на ключ в минуту),
  а устаревший кэш ломает только UX — двойную бронь всё равно отсекает PK `bookings` при `Approve`.
  Услугам можно дольше, но раздельный TTL — правка `config.go` (общая зона). Вариант для
  исследования: прогнать нагрузку с разным `CACHE_TTL`, сравнить hit rate и запросы в Postgres.
- **Кэш расписания:** кэшировать и пустые дни (`[]`), иначе свободные дни всегда промах. Отличать
  `redis.Nil` от закэшированного пустого списка (nil-срез сериализуется в `null`). `DEL` — только
  после успешного `Approve`. Гонка «промах → чтение БД → Approve+DEL → устаревший SET» ограничена
  `CACHE_TTL` — описать в исследовании, не чинить.

### Исследование
- Перед замерами очистить Redis (`make flush` или `docker compose down -v`): в volume лежат ключи
  старой схемы (`request:*`, `service:*`, `services:index`, `requests:*`) без TTL — под
  `volatile-lru` они не вытесняются и портят картину.
- «Рестарт без AOF»: `docker stop` — это штатный `SHUTDOWN`, при `save 60 100` Redis пишет RDB и всё
  восстановится. Отдельно гонять с `docker kill` (SIGKILL).
- TTL хранится как абсолютное время истечения — простой съедает TTL.
- `make keys` делает 2 `docker exec` на ключ — на 64 МБ непригоден; смотреть `INFO keyspace`,
  `INFO stats` (`evicted_keys`, `expired_keys`).
- Слишком маленький `CART_TTL` (< 1 мс) превращает `PEXPIRE` в `PEXPIRE 0` — корзина удаляется.

## Заметки для Б

- `deploy/postgres/init.sql`: в `users` колонки `email`, `name`, а в домене `user.User` — `Login`.
- `services.kind` допускает `consultation`, которого нет в `service.Kind`.
- Общая зона, `internal/config/config.go`: `envInt`/`envDuration` молча берут дефолт при битом
  значении (`CART_TTL=10` без единиц → 30m); `validate` проверяет только `> 0`. Предложение: ошибка,
  если переменная задана, но не парсится; TTL и окна `>= 1s`.
- Общая зона, `config.go:45`: `godotenv.Load("deploy/.env", ".env")` останавливается на первом
  отсутствующем файле — `.env` без `deploy/.env` не грузится.
- Общая зона, домен: `cart.ErrInvalidUser` не используется и не в `errorMap` (→ 500);
  `cart.New(string)` и код `invalid_id` не из CONTRACTS; `Cart.UpdatedAt` Redis не хранит (после
  `Get` — нулевое время); `user.ErrInvalidCredentials` говорит «email», а в домене login.
