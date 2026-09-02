package domain

import (
	"time"

	"github.com/google/uuid"
)

type Company struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Website     *string   `json:"website,omitempty"`
	Description *string   `json:"description,omitempty"`
	LogoURL     *string   `json:"logo_url,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
