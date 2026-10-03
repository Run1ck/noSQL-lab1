package request

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// Create сохраняет новую заявку и записывает в r.ID выданный базой id.
	Create(ctx context.Context, r *Request) error
	// Get возвращает ErrNotFound, если заявки нет.
	Get(ctx context.Context, id int64) (*Request, error)
	// List, ListByStatus и ListByUser отдают заявки вместе с позициями,
	// новые сверху.
	List(ctx context.Context) ([]*Request, error)
	ListByStatus(ctx context.Context, s Status) ([]*Request, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Request, error)

	// Approve, Reject и Cancel сохраняют заявку после одноимённого доменного
	// метода. Запись идёт условным UPDATE ... WHERE id = $1 AND status = 'new':
	// если заявку уже обработали параллельно (0 строк) — ErrAlreadyProcessed.

	// Approve в той же транзакции записывает дни заявки в bookings — день
	// занимает только одобрение. Если день уже занят другой одобренной
	// заявкой — ErrSlotBooked, и заявка остаётся new.
	Approve(ctx context.Context, r *Request) error
	Reject(ctx context.Context, r *Request) error
	Cancel(ctx context.Context, r *Request) error
}
