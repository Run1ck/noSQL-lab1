package postgres

import (
	"booking/internal/domain/service"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Коды SQLSTATE, которые репозитории переводят в доменные ошибки.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// isPgCode сообщает, вернул ли Postgres ошибку с этим кодом SQLSTATE.
func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// dateToDB переводит день в значение для колонки DATE. pgx пишет из
// time.Time только год, месяц и день, а читает дату как полночь UTC —
// UTC здесь для симметрии с чтением.
func dateToDB(d service.Date) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}
