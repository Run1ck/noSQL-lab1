// Seeder наполняет базу тестовыми данными: услуги и несколько одобренных
// заявок на ближайшие дни, чтобы в расписании были занятые дни. Повторный
// запуск безопасен: услуги обновляются, занятые дни пропускаются.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"booking/config"
	adapterpostgres "booking/internal/adapter/postgres"
	"booking/internal/domain/request"
	"booking/internal/domain/service"
	"booking/internal/domain/user"
	"booking/pkg/postgres"
)

func main() {
	c, err := config.Load()
	if err != nil {
		fail("config.Load", err)
	}

	ctx := context.Background()

	pool, err := postgres.New(ctx, c.PostgresDSN)
	if err != nil {
		fail("postgres.New", err)
	}
	defer pool.Close()

	pg := adapterpostgres.New(pool)

	services := Services()
	for _, s := range services {
		err = pg.SaveService(ctx, s)
		if err != nil {
			fail("pg.SaveService", err)
		}
	}

	student, err := ensureUser(ctx, pg, "student", user.RoleUser)
	if err != nil {
		fail("ensureUser student", err)
	}

	admin, err := ensureUser(ctx, pg, "seed-admin", user.RoleAdmin)
	if err != nil {
		fail("ensureUser seed-admin", err)
	}

	approved := 0

	for _, items := range Bookings(time.Now()) {
		ok, err := approve(ctx, pg, student.ID, admin.ID, items)
		if err != nil {
			fail("approve", err)
		}

		if ok {
			approved++
		}
	}

	fmt.Printf("seeded: %d services, %d new approved requests\n", len(services), approved)
}

// Services — аудитории, лаборатории и оборудование; две услуги неактивны.
func Services() []*service.Service {
	var out []*service.Service

	add := func(id, name string, kind service.Kind, active bool) {
		out = append(out, &service.Service{ID: id, Name: name, Kind: kind, Active: active})
	}

	for i := 101; i <= 110; i++ {
		add(fmt.Sprintf("room-%d", i), fmt.Sprintf("Аудитория %d", i), service.Room, true)
	}

	for i := 1; i <= 5; i++ {
		add(fmt.Sprintf("lab-%d", i), fmt.Sprintf("Лаборатория %d", i), service.Lab, i != 5)
	}

	for i := 1; i <= 5; i++ {
		add(fmt.Sprintf("projector-%d", i), fmt.Sprintf("Проектор %d", i), service.Equipment, i != 5)
	}

	return out
}

// Bookings — позиции заявок на ближайшие дни; каждая заявка будет одобрена.
func Bookings(now time.Time) [][]request.Item {
	day := func(offset int) service.Date {
		return service.DateOf(now.AddDate(0, 0, offset))
	}

	return [][]request.Item{
		{{ServiceID: "room-101", Date: day(1)}, {ServiceID: "lab-1", Date: day(1)}},
		{{ServiceID: "room-102", Date: day(2)}},
		{{ServiceID: "room-101", Date: day(3)}, {ServiceID: "projector-1", Date: day(3)}},
	}
}

func ensureUser(ctx context.Context, pg *adapterpostgres.Postgres, login string, role user.Role) (*user.User, error) {
	u, err := pg.GetUserByLogin(ctx, login)
	if err == nil {
		return u, nil
	}

	if !errors.Is(err, user.ErrNotFound) {
		return nil, fmt.Errorf("pg.GetUserByLogin: %w", err)
	}

	u, err = user.New(login, "seed-password", role)
	if err != nil {
		return nil, fmt.Errorf("user.New: %w", err)
	}

	err = pg.CreateUser(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("pg.CreateUser: %w", err)
	}

	return u, nil
}

// approve создаёт заявку и одобряет её. Если какой-то день уже занят
// (повторный запуск), заявку не создаёт.
func approve(ctx context.Context, pg *adapterpostgres.Postgres, userID, adminID uuid.UUID, items []request.Item) (bool, error) {
	for _, item := range items {
		bookings, err := pg.GetBookings(ctx, item.Date)
		if err != nil {
			return false, fmt.Errorf("pg.GetBookings: %w", err)
		}

		for _, b := range bookings {
			if b.ServiceID == item.ServiceID {
				return false, nil
			}
		}
	}

	r := &request.Request{UserID: userID, Items: items, Status: request.StatusNew, CreatedAt: time.Now()}

	err := pg.CreateRequest(ctx, r)
	if err != nil {
		return false, fmt.Errorf("pg.CreateRequest: %w", err)
	}

	err = r.Approve(adminID, "seed")
	if err != nil {
		return false, fmt.Errorf("r.Approve: %w", err)
	}

	err = pg.Approve(ctx, r)
	if errors.Is(err, request.ErrSlotBooked) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("pg.Approve: %w", err)
	}

	return true, nil
}

func fail(msg string, err error) {
	slog.Error(msg, "err", err)
	os.Exit(1)
}
