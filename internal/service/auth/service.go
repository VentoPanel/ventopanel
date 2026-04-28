package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	domain "github.com/your-org/ventopanel/internal/domain/user"
)

const bcryptCost = 12

type Service struct {
	repo        domain.Repository
	jwtSecret   string
	jwtIssuer   string
	jwtAudience string
	tokenTTL    time.Duration
}

func NewService(
	repo domain.Repository,
	jwtSecret, jwtIssuer, jwtAudience string,
	tokenTTL time.Duration,
) *Service {
	return &Service{
		repo:        repo,
		jwtSecret:   jwtSecret,
		jwtIssuer:   jwtIssuer,
		jwtAudience: jwtAudience,
		tokenTTL:    tokenTTL,
	}
}

// Register creates a new user. The first registered user always gets the admin role.
// Subsequent registrations require an existing admin to call this endpoint.
func (s *Service) Register(ctx context.Context, email, password, teamID string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" || teamID == "" {
		return nil, fmt.Errorf("email, password and team_id are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}

	role := domain.RoleViewer
	if count == 0 {
		role = domain.RoleAdmin
	}

	u := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		TeamID:       teamID,
		Role:         role,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login verifies credentials and returns a signed JWT.
func (s *Service) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, domain.ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, domain.ErrInvalidCreds
	}

	token, err := s.issueToken(u)
	if err != nil {
		return "", nil, fmt.Errorf("issue token: %w", err)
	}

	return token, u, nil
}

func (s *Service) issueToken(u *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":     u.ID,
		"tid":     u.TeamID,
		"role":    u.Role,
		"email":   u.Email,
		"iat":     now.Unix(),
		"nbf":     now.Unix(),
		"exp":     now.Add(s.tokenTTL).Unix(),
	}
	if s.jwtIssuer != "" {
		claims["iss"] = s.jwtIssuer
	}
	if s.jwtAudience != "" {
		claims["aud"] = s.jwtAudience
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.jwtSecret))
}
