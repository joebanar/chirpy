package database

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
}
