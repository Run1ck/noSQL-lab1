package redisstore

import (
	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Корзина — SET cart:{userID} из позиций "serviceID|YYYY-MM-DD" с одним TTL
// на всю корзину. Пустой SET Redis удаляет сам, поэтому «корзины нет»
// и «корзина пуста» неразличимы.

// removeItemScript удаляет позицию и продлевает TTL одной атомарной операцией.
//
//	KEYS[1] — ключ корзины, cart:{userID}
//	ARGV[1] — позиция, "serviceID|YYYY-MM-DD"
//	ARGV[2] — TTL корзины в миллисекундах
//
// Возвращает 1, если позиция удалена; 0, если её не было — тогда TTL не трогает.
var removeItemScript = redis.NewScript(`
if redis.call('SREM', KEYS[1], ARGV[1]) == 0 then
	return 0
end
redis.call('PEXPIRE', KEYS[1], ARGV[2])
return 1
`)

// Carts — cart.Repository на Redis.
type Carts struct {
	rdb *redis.Client
	// ttl — время жизни корзины (CART_TTL), отсчитывается заново при каждом изменении.
	ttl time.Duration
}

// NewCarts возвращает репозиторий корзин с временем жизни ttl.
func NewCarts(rdb *redis.Client, ttl time.Duration) *Carts {
	return &Carts{rdb: rdb, ttl: ttl}
}

func (r *Carts) Get(ctx context.Context, userID uuid.UUID) (*cart.Cart, error) {
	members, err := r.rdb.SMembers(ctx, cartKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}
	c := &cart.Cart{
		UserID: userID,
		Items:  make(map[cart.Item]struct{}, len(members)),
	}
	for _, m := range members {
		item, err := decodeItem(m)
		if err != nil {
			return nil, err
		}
		c.Items[item] = struct{}{}
	}
	return c, nil
}

// Save перезаписывает корзину целиком: DEL, SADD и PEXPIRE в одной
// транзакции MULTI/EXEC, чтобы корзина не осталась без TTL.
func (r *Carts) Save(ctx context.Context, c *cart.Cart) error {
	key := cartKey(c.UserID)
	if c.IsEmpty() {
		return r.DeleteCart(ctx, c.UserID)
	}
	members := make([]any, 0, len(c.Items))
	for item := range c.Items {
		// Битая позиция не прочитается обратно, и Get будет падать до конца TTL.
		if item.ServiceID == "" || !item.Date.IsValid() {
			return fmt.Errorf("save cart: invalid item %+v", item)
		}
		members = append(members, encodeItem(item))
	}
	_, err := r.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, key)
		p.SAdd(ctx, key, members...)
		p.PExpire(ctx, key, r.ttl)
		return nil
	})
	if err != nil {
		return fmt.Errorf("save cart: %w", err)
	}
	return nil
}

func (r *Carts) RemoveItem(ctx context.Context, userID uuid.UUID, item cart.Item) error {
	removed, err := removeItemScript.Run(ctx, r.rdb,
		[]string{cartKey(userID)}, encodeItem(item), r.ttl.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("remove cart item: %w", err)
	}
	if removed == 0 {
		return cart.ErrItemNotFound
	}
	return nil
}

func (r *Carts) DeleteCart(ctx context.Context, userID uuid.UUID) error {
	if err := r.rdb.Del(ctx, cartKey(userID)).Err(); err != nil {
		return fmt.Errorf("delete cart: %w", err)
	}
	return nil
}

func (r *Carts) TTL(ctx context.Context, userID uuid.UUID) (time.Duration, error) {
	d, err := r.rdb.PTTL(ctx, cartKey(userID)).Result()
	if err != nil {
		return 0, fmt.Errorf("cart ttl: %w", err)
	}
	// -2 — ключа нет, -1 — ключ без TTL; такой корзину не пишем.
	if d < 0 {
		return 0, nil
	}
	return d, nil
}

func cartKey(userID uuid.UUID) string {
	return "cart:" + userID.String()
}

func encodeItem(item cart.Item) string {
	return item.ServiceID + "|" + item.Date.String()
}

// decodeItem режет по последнему '|': в дате его нет, а в ID услуги может быть.
func decodeItem(s string) (cart.Item, error) {
	i := strings.LastIndexByte(s, '|')
	if i < 0 {
		return cart.Item{}, fmt.Errorf("bad cart item %q", s)
	}
	date, err := service.ParseDate(s[i+1:])
	if err != nil {
		// %v, а не %w: битые данные в Redis — это 500, а не 400 invalid_date.
		return cart.Item{}, fmt.Errorf("bad cart item %q: %v", s, err)
	}
	return cart.Item{ServiceID: s[:i], Date: date}, nil
}
