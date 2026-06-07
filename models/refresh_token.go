package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken stores long-lived tokens for sliding sessions.
type RefreshToken struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;index;not null"`
	Token     string    `json:"token" gorm:"size:255;uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationship
	User User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
