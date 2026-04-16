package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User stores account data used for authentication and ownership.
type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Email        string    `json:"email" gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	return nil
}
