package usecase

import (
	"context"
	"fmt"
	"time"

	"booking/internal/domain/cart"
	"booking/internal/domain/service"
	"booking/internal/dto"
)

func (u *UseCase) AddCartItem(ctx context.Context, input dto.AddCartItemInput) (dto.CartOutput, error) {
	var output dto.CartOutput

	date, err := service.ParseDate(input.Date)
	if err != nil {
		return output, fmt.Errorf("service.ParseDate: %w", err)
	}

	svc, err := u.postgres.GetService(ctx, input.ServiceID)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetService: %w", err)
	}

	if !svc.Active {
		return output, cart.ErrServiceUnavailable
	}

	if compareDates(date, service.DateOf(time.Now())) < 0 {
		return output, cart.ErrPastDate
	}

	bookings, err := u.postgres.GetBookings(ctx, date)
	if err != nil {
		return output, fmt.Errorf("u.postgres.GetBookings: %w", err)
	}

	for _, b := range bookings {
		if b.ServiceID == svc.ID {
			return output, cart.ErrSlotBooked
		}
	}

	c, err := u.redis.GetCart(ctx, input.UserID)
	if err != nil {
		return output, fmt.Errorf("u.redis.GetCart: %w", err)
	}

	c.AddItem(cart.Item{ServiceID: svc.ID, Date: date})

	err = u.redis.SaveCart(ctx, c)
	if err != nil {
		return output, fmt.Errorf("u.redis.SaveCart: %w", err)
	}

	output, err = u.cart(ctx, input.UserID)
	if err != nil {
		return output, fmt.Errorf("u.cart: %w", err)
	}

	return output, nil
}
