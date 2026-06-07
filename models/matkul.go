package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Matkul stores user-owned subject/course data.
type Matkul struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_matkul_user_name,priority:1"`
	Name      string    `json:"name" gorm:"size:120;not null;uniqueIndex:idx_matkul_user_name,priority:2"`
	Code      string    `json:"code" gorm:"size:30"`
	Semester  string    `json:"semester" gorm:"size:20"`
	Tag       string    `json:"tag" gorm:"size:50"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User User `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (m *Matkul) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (m *Matkul) BeforeSave(tx *gorm.DB) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Code = strings.TrimSpace(m.Code)
	m.Semester = strings.TrimSpace(m.Semester)
	m.Tag = strings.TrimSpace(m.Tag)
	return nil
}
