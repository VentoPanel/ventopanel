package user

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("user not found")
	ErrEmailTaken     = errors.New("email already registered")
	ErrInvalidCreds   = errors.New("invalid email or password")
)

// Role values.
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	TeamID       string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Count(ctx context.Context) (int64, error)
}
