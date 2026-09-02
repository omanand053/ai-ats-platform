package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"ai-ats-platform/backend/internal/auth"
	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrSlugAlreadyExists  = errors.New("company slug already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountInactive    = errors.New("account is inactive")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type SignupInput struct {
	CompanyName string
	CompanySlug string
	Email       string
	Password    string
	FirstName   string
	LastName    string
}

type AuthResult struct {
	User        *domain.User    `json:"user"`
	Company     *domain.Company `json:"company,omitempty"`
	AccessToken string          `json:"access_token"`
	TokenType   string          `json:"token_type"`
	ExpiresIn   int64           `json:"expires_in"`
}

type AuthService struct {
	pool        *pgxpool.Pool
	users       *repository.UserRepository
	companies   *repository.CompanyRepository
	tokenMgr    *auth.TokenManager
	tokenExpiry int64
}

func NewAuthService(
	pool *pgxpool.Pool,
	users *repository.UserRepository,
	companies *repository.CompanyRepository,
	tokenMgr *auth.TokenManager,
	tokenExpiry int64,
) *AuthService {
	return &AuthService{
		pool:        pool,
		users:       users,
		companies:   companies,
		tokenMgr:    tokenMgr,
		tokenExpiry: tokenExpiry,
	}
}

func (s *AuthService) Signup(ctx context.Context, input SignupInput) (*AuthResult, error) {
	slug := input.CompanySlug
	if slug == "" {
		slug = generateSlug(input.CompanyName)
	}
	if !slugPattern.MatchString(slug) {
		return nil, fmt.Errorf("invalid company slug format")
	}

	emailExists, err := s.users.EmailExists(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, ErrEmailAlreadyExists
	}

	slugExists, err := s.companies.SlugExists(ctx, slug)
	if err != nil {
		return nil, err
	}
	if slugExists {
		return nil, ErrSlugAlreadyExists
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	company := &domain.Company{
		Name: input.CompanyName,
		Slug: slug,
	}

	if err := s.companies.Create(ctx, tx, company); err != nil {
		return nil, fmt.Errorf("create company: %w", err)
	}

	user := &domain.User{
		CompanyID:    company.ID,
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(input.FirstName),
		LastName:     strings.TrimSpace(input.LastName),
		Role:         "admin",
	}

	if err := s.users.Create(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	token, err := s.tokenMgr.Generate(user.ID, user.CompanyID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{
		User:        user,
		Company:     company,
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   s.tokenExpiry,
	}, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	if !auth.CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Printf("warning: update last login for %s: %v", user.ID, err)
	}

	token, err := s.tokenMgr.Generate(user.ID, user.CompanyID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{
		User:        user,
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   s.tokenExpiry,
	}, nil
}

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "company"
	}
	return slug
}
