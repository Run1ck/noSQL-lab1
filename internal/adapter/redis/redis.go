package redis

import (
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"booking/internal/domain/cart"
)

// Корзина — SET cart:{userID} из позиций "serviceID|YYYY-MM-DD" с одним TTL на
// всю корзину. Пустой SET Redis удаляет сам: «корзины нет» и «корзина пуста»
// неразличимы.
const cartPrefix = "cart:"

type Redis struct {
	redis   *redis.Client
	cartTTL time.Duration
}

func New(client *redis.Client, cartTTL time.Duration) *Redis {
	return &Redis{
		redis:   client,
		cartTTL: cartTTL,
	}
}

func cartKey(userID uuid.UUID) string {
	return cartPrefix + userID.String()
}

func encodeItem(item cart.Item) string {
	return item.ServiceID + "|" + item.Date.String()
}
