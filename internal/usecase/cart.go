package usecase

import (
	"booking/internal/domain/booking"
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
)

// CartView — корзина глазами пользователя: позиции по дате, затем по ID
// услуги; ExpiresIn — сколько корзине осталось жить, 0 — корзины нет.
type CartView struct {
	Items     []cart.Item
	ExpiresIn time.Duration
}

// Cart — временная корзина пользователя.
type Cart struct {
	carts    cart.Repository
	services service.Repository
	bookings booking.Repository
	// now — часы для правила «не раньше сегодня»; тесты подменяют их.
	now func() time.Time
}

// NewCart принимает корзины (Redis) и услуги с бронями — в main это
// кэш-декораторы поверх Postgres: по ним проверяется, можно ли положить
// позицию в корзину.
func NewCart(carts cart.Repository, services service.Repository, bookings booking.Repository) *Cart {
	return &Cart{carts: carts, services: services, bookings: bookings, now: time.Now}
}

func (uc *Cart) Get(ctx context.Context, userID uuid.UUID) (CartView, error) {
	return uc.view(ctx, userID)
}

// AddItem кладёт позицию в корзину. Проверки — в порядке кодов ошибок
// контракта: услуга есть, активна, день не раньше сегодняшнего (по часам
// сервера), не занят бронью этой услуги. Повторное добавление той же пары —
// не ошибка, только продлевает TTL.
func (uc *Cart) AddItem(ctx context.Context, userID uuid.UUID, item cart.Item) (CartView, error) {
	svc, err := uc.services.Get(ctx, item.ServiceID)
	if err != nil {
		return CartView{}, err
	}
	if !svc.Active {
		return CartView{}, cart.ErrServiceUnavailable
	}
	if compareDates(item.Date, service.DateOf(uc.now())) < 0 {
		return CartView{}, cart.ErrPastDate
	}
	bookings, err := uc.bookings.ListByDate(ctx, item.Date)
	if err != nil {
		return CartView{}, err
	}
	for _, b := range bookings {
		if b.ServiceID == item.ServiceID {
			return CartView{}, cart.ErrSlotBooked
		}
	}

	// Get → AddItem → Save: два параллельных добавления могут потерять
	// позицию — гонка принята (docs/PART_A.md, «Корзина»).
	c, err := uc.carts.Get(ctx, userID)
	if err != nil {
		return CartView{}, err
	}
	c.AddItem(item)
	if err := uc.carts.Save(ctx, c); err != nil {
		return CartView{}, err
	}
	return uc.view(ctx, userID)
}

// RemoveItem убирает позицию; если её нет — cart.ErrItemNotFound.
func (uc *Cart) RemoveItem(ctx context.Context, userID uuid.UUID, item cart.Item) (CartView, error) {
	if err := uc.carts.RemoveItem(ctx, userID, item); err != nil {
		return CartView{}, err
	}
	return uc.view(ctx, userID)
}

// Clear удаляет корзину; пустая корзина — не ошибка.
func (uc *Cart) Clear(ctx context.Context, userID uuid.UUID) error {
	return uc.carts.DeleteCart(ctx, userID)
}

func (uc *Cart) view(ctx context.Context, userID uuid.UUID) (CartView, error) {
	c, err := uc.carts.Get(ctx, userID)
	if err != nil {
		return CartView{}, err
	}
	ttl, err := uc.carts.TTL(ctx, userID)
	if err != nil {
		return CartView{}, err
	}
	items := make([]cart.Item, 0, len(c.Items))
	for it := range c.Items {
		items = append(items, it)
	}
	slices.SortFunc(items, func(a, b cart.Item) int {
		return cmp.Or(compareDates(a.Date, b.Date), cmp.Compare(a.ServiceID, b.ServiceID))
	})
	return CartView{Items: items, ExpiresIn: ttl}, nil
}
