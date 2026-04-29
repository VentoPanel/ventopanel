package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
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

// LoginResult is returned by Login — either a full token or an MFA challenge.
type LoginResult struct {
	Token       string // non-empty when login is complete (no 2FA, or 2FA already verified)
	MFARequired bool   // true when the user has 2FA enabled
	MFASession  string // short-lived token to complete MFA step
	User        *domain.User
}

// Login verifies credentials. If the user has 2FA enabled it returns an MFA session
// instead of a full token; the caller must complete the MFA step.
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return LoginResult{}, domain.ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, domain.ErrInvalidCreds
	}

	if u.TOTPEnabled {
		mfaSession, err := s.issueMFASession(u)
		if err != nil {
			return LoginResult{}, fmt.Errorf("issue mfa session: %w", err)
		}
		return LoginResult{MFARequired: true, MFASession: mfaSession, User: u}, nil
	}

	token, err := s.issueToken(u)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue token: %w", err)
	}
	return LoginResult{Token: token, User: u}, nil
}

// VerifyMFA validates a TOTP code against an MFA session and returns a full JWT on success.
func (s *Service) VerifyMFA(ctx context.Context, mfaSession, code string) (string, *domain.User, error) {
	claims, err := s.parseMFASession(mfaSession)
	if err != nil {
		return "", nil, fmt.Errorf("invalid mfa session")
	}
	userID, _ := claims["uid"].(string)
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return "", nil, domain.ErrInvalidCreds
	}
	if !u.TOTPEnabled || u.TOTPSecret == "" {
		return "", nil, fmt.Errorf("2FA not enabled for this user")
	}
	if !totp.Validate(code, u.TOTPSecret) {
		return "", nil, fmt.Errorf("invalid 2FA code")
	}
	token, err := s.issueToken(u)
	if err != nil {
		return "", nil, fmt.Errorf("issue token: %w", err)
	}
	return token, u, nil
}

// SetupTOTP generates a new TOTP secret for a user (does not enable it yet).
// Returns the base32 secret and an otpauth:// URL for QR code generation.
func (s *Service) SetupTOTP(ctx context.Context, userID string) (secret, otpauthURL string, err error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return "", "", err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VentoPanel",
		AccountName: u.Email,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate totp: %w", err)
	}
	// Save the pending secret (not yet enabled).
	if err := s.repo.UpdateTOTP(ctx, userID, key.Secret(), false); err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// EnableTOTP verifies a TOTP code against the pending secret and enables 2FA.
func (s *Service) EnableTOTP(ctx context.Context, userID, code string) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.TOTPSecret == "" {
		return fmt.Errorf("run setup first")
	}
	if !totp.Validate(code, u.TOTPSecret) {
		return fmt.Errorf("invalid code")
	}
	return s.repo.UpdateTOTP(ctx, userID, u.TOTPSecret, true)
}

// DisableTOTP verifies a TOTP code and disables 2FA.
func (s *Service) DisableTOTP(ctx context.Context, userID, code string) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !u.TOTPEnabled {
		return fmt.Errorf("2FA is not enabled")
	}
	if !totp.Validate(code, u.TOTPSecret) {
		return fmt.Errorf("invalid code")
	}
	return s.repo.UpdateTOTP(ctx, userID, "", false)
}

// issueMFASession returns a short-lived JWT (5 min) with mfa:pending claim.
func (s *Service) issueMFASession(u *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":     u.ID,
		"mfa":     "pending",
		"iat":     now.Unix(),
		"exp":     now.Add(5 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.jwtSecret))
}

// parseMFASession validates and returns claims from an MFA session token.
func (s *Service) parseMFASession(tokenStr string) (jwt.MapClaims, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	if claims["mfa"] != "pending" {
		return nil, fmt.Errorf("not an mfa session")
	}
	return claims, nil
}

func (s *Service) issueToken(u *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":          u.ID,
		"tid":          u.TeamID,
		"role":         u.Role,
		"email":        u.Email,
		"totp_enabled": u.TOTPEnabled,
		"iat":          now.Unix(),
		"nbf":          now.Unix(),
		"exp":          now.Add(s.tokenTTL).Unix(),
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
