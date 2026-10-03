package cart

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository хранит корзины во временном хранилище: у корзины есть TTL
// (CART_TTL), и каждое изменение продлевает его заново.
type Repository interface {
	// Get возвращает корзину пользователя. Если корзины нет — не создавали
	// или истёк TTL, — возвращает пустую корзину, а не ошибку.
	Get(ctx context.Context, userID uuid.UUID) (*Cart, error)
	// Save сохраняет корзину и продлевает TTL. Пустая корзина удаляется.
	Save(ctx context.Context, c *Cart) error

	// RemoveItem удаляет позицию и продлевает TTL; если позиции нет —
	// ErrItemNotFound.
	RemoveItem(ctx context.Context, userID uuid.UUID, item Item) error
	// DeleteCart удаляет корзину; если её нет — не ошибка.
	DeleteCart(ctx context.Context, userID uuid.UUID) error

	// TTL возвращает, сколько корзине осталось жить; 0 — корзины нет.
	TTL(ctx context.Context, userID uuid.UUID) (time.Duration, error)
}
