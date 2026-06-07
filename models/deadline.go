package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Deadline stores user-owned assignments linked to a specific course.
type Deadline struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index:idx_deadline_user_due,priority:1"`
	MatkulID    uuid.UUID `json:"matkul_id" gorm:"type:uuid;not null;index"`
	Title       string    `json:"title" gorm:"size:160;not null"`
	Description string    `json:"description" gorm:"size:1000"`
	DueAt       time.Time `json:"due_at" gorm:"not null;index:idx_deadline_user_due,priority:2"`
	Status                string    `json:"status" gorm:"size:20;not null;default:todo;index"`
	Priority              int       `json:"priority" gorm:"not null;default:0"`
	ReminderOffsetMinutes int       `json:"reminder_offset_minutes" gorm:"not null;default:60"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	User   User   `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Matkul Matkul `json:"-" gorm:"foreignKey:MatkulID;references:ID;constraint:OnDelete:CASCADE"`
}

func (d *Deadline) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

func (d *Deadline) BeforeSave(tx *gorm.DB) error {
	d.Title = strings.TrimSpace(d.Title)
	d.Description = strings.TrimSpace(d.Description)
	d.Status = strings.ToLower(strings.TrimSpace(d.Status))
	if d.Status == "" {
		d.Status = "todo"
	}
	return nil
}
