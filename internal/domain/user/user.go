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
	ErrWrongPassword  = errors.New("current password is incorrect")
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
	TOTPSecret   string
	TOTPEnabled  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Count(ctx context.Context) (int64, error)
	List(ctx context.Context) ([]User, error)
	UpdateRole(ctx context.Context, id, role string) error
	UpdateTOTP(ctx context.Context, id, secret string, enabled bool) error
	UpdatePassword(ctx context.Context, id, newHash string) error
	UpdateEmail(ctx context.Context, id, email string) error
	Delete(ctx context.Context, id string) error
}
