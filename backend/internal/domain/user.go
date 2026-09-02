package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	CompanyID       uuid.UUID  `json:"company_id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	Role            string     `json:"role"`
	IsActive        bool       `json:"is_active"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
