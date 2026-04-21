package repository

import (
	"context"
	"mini-project/internal/model"
)

type UserRepository interface {
	//Write
	CreateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id string) error

	//Credit
	AddCredit(ctx context.Context, id string, amount float64) error
	DeductCredit(ctx context.Context, id string, amount float64) error
	Transfer(ctx context.Context, senderId string, receiverId string, amount float64) error

	//Read
	GetAll(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
	GetCreditByID(ctx context.Context, id string) (float64, error)
	GetByUsername(ctx context.Context, username string) (model.User, error)

	//Clear User-related cache
	ClearCache(ctx context.Context, id string)
}