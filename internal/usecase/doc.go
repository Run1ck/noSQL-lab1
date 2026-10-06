// Package usecase — сценарии приложения: что можно сделать и по каким
// правилам. Use case знает домен и интерфейсы репозиториев, но не HTTP,
// не Redis и не Postgres: его вызывает delivery (internal/api), а адаптеры
// (storage, cache, ratelimit) подставляет main.
//
// Файлы по владельцам: catalog, schedule, cart, dates — А; auth, requests,
// admin — Б.
package usecase
