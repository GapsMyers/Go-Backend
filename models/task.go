package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Task stores assignments linked to a specific matkul.
type Task struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Title     string    `json:"title" gorm:"not null"`
	MatkulID  uuid.UUID `json:"matkul_id" gorm:"type:uuid;not null;index"`
	Priority  string    `json:"priority" gorm:"size:10;not null;default:medium"`
	IsDone    bool      `json:"is_done" gorm:"default:false;index"`
	Deadline  time.Time `json:"deadline"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Matkul Matkul `json:"-" gorm:"foreignKey:MatkulID;references:ID;constraint:OnDelete:CASCADE"`
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (t *Task) BeforeSave(tx *gorm.DB) error {
	t.Title = strings.TrimSpace(t.Title)
	t.Priority = strings.ToLower(strings.TrimSpace(t.Priority))
	if t.Priority == "" {
		t.Priority = "medium"
	}
	return nil
}
