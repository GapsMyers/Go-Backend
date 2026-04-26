package handlers

import (
	"backend/middleware"
	"backend/models"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeadlineHandler struct {
	DB *gorm.DB
}

func NewDeadlineHandler(db *gorm.DB) *DeadlineHandler {
	return &DeadlineHandler{DB: db}
}

type createDeadlineRequest struct {
	MatkulID    string `json:"matkul_id" binding:"required,uuid"`
	Title       string `json:"title" binding:"required,min=2,max=160"`
	Description string `json:"description" binding:"max=1000"`
	DueAt                 string `json:"due_at" binding:"required,datetime=2006-01-02T15:04:05Z07:00"`
	Priority              int    `json:"priority"`
	ReminderOffsetMinutes int    `json:"reminder_offset_minutes" binding:"required"`
}

type updateDeadlineRequest struct {
	Title                 string `json:"title" binding:"omitempty,min=2,max=160"`
	Description           string `json:"description" binding:"max=1000"`
	DueAt                 string `json:"due_at" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Priority              *int   `json:"priority"`
	ReminderOffsetMinutes *int   `json:"reminder_offset_minutes"`
}

type deadlineResponse struct {
	ID          string `json:"id"`
	MatkulID    string `json:"matkul_id"`
	MatkulName  string `json:"matkul_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueAt                 string `json:"due_at"`
	Status                string `json:"status"`
	Priority              int    `json:"priority"`
	ReminderOffsetMinutes int    `json:"reminder_offset_minutes"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

func (h *DeadlineHandler) Create(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	var req createDeadlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid deadline payload", err.Error())
		return
	}

	matkulID, err := uuid.Parse(req.MatkulID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "matkul_id must be a valid uuid", nil)
		return
	}

	dueAt, err := time.Parse(time.RFC3339, req.DueAt)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "due_at must use RFC3339 format", nil)
		return
	}
	if dueAt.Before(time.Now().Add(-1 * time.Minute)) {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "due_at must be in the future", nil)
		return
	}

	if req.ReminderOffsetMinutes != 60 && req.ReminderOffsetMinutes != 180 && req.ReminderOffsetMinutes != 1440 && req.ReminderOffsetMinutes != 4320 {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "reminder_offset_minutes must be one of: 60, 180, 1440, 4320", nil)
		return
	}

	var matkul models.Matkul
	if err := h.DB.Where("id = ? AND user_id = ?", matkulID, userID).First(&matkul).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "MATKUL_NOT_FOUND", "matkul not found for this user", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to validate matkul", nil)
		return
	}

	deadline := models.Deadline{
		UserID:      userID,
		MatkulID:    matkulID,
		Title:                 strings.TrimSpace(req.Title),
		Description:           strings.TrimSpace(req.Description),
		DueAt:                 dueAt.UTC(),
		Status:                "todo",
		Priority:              req.Priority,
		ReminderOffsetMinutes: req.ReminderOffsetMinutes,
	}

	if err := h.DB.Create(&deadline).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create deadline", nil)
		return
	}

	writeSuccess(c, http.StatusCreated, "deadline created", toDeadlineResponse(deadline, matkul.Name))
}

func (h *DeadlineHandler) List(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	query := h.DB.Preload("Matkul").Where("user_id = ?", userID)

	if status := strings.ToLower(strings.TrimSpace(c.Query("status"))); status != "" {
		query = query.Where("status = ?", status)
	}

	if matkulParam := strings.TrimSpace(c.Query("matkul_id")); matkulParam != "" {
		matkulID, err := uuid.Parse(matkulParam)
		if err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "matkul_id query must be a valid uuid", nil)
			return
		}
		query = query.Where("matkul_id = ?", matkulID)
	}

	var deadlines []models.Deadline
	if err := query.Order("due_at ASC").Find(&deadlines).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deadlines", nil)
		return
	}

	response := make([]deadlineResponse, 0, len(deadlines))
	for _, deadline := range deadlines {
		response = append(response, toDeadlineResponse(deadline, deadline.Matkul.Name))
	}

	writeSuccess(c, http.StatusOK, "deadline list fetched", response)
}

func toDeadlineResponse(deadline models.Deadline, matkulName string) deadlineResponse {
	return deadlineResponse{
		ID:          deadline.ID.String(),
		MatkulID:    deadline.MatkulID.String(),
		MatkulName:  matkulName,
		Title:       deadline.Title,
		Description: deadline.Description,
		DueAt:                 deadline.DueAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Status:                deadline.Status,
		Priority:              deadline.Priority,
		ReminderOffsetMinutes: deadline.ReminderOffsetMinutes,
		CreatedAt:             deadline.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:             deadline.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (h *DeadlineHandler) Update(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	deadlineID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid deadline id", nil)
		return
	}

	var deadline models.Deadline
	if err := h.DB.Preload("Matkul").Where("id = ? AND user_id = ?", deadlineID, userID).First(&deadline).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "deadline not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to find deadline", nil)
		return
	}

	var req updateDeadlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid deadline update payload", err.Error())
		return
	}

	if req.ReminderOffsetMinutes != nil {
		rm := *req.ReminderOffsetMinutes
		if rm != 60 && rm != 180 && rm != 1440 && rm != 4320 {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "reminder_offset_minutes must be one of: 60, 180, 1440, 4320", nil)
			return
		}
		deadline.ReminderOffsetMinutes = rm
	}

	if title := strings.TrimSpace(req.Title); title != "" {
		deadline.Title = title
	}
	// Description can be empty
	if req.Description != "" || strings.TrimSpace(req.Description) == "" {
		// Only update if it's explicitly set? With binding we check if we should update.
		// Actually the existing Matkul approach just updates it if it's there. Since description isn't omitempty, 
		// we'll check if json has the key or just simply update it. Let's just update. 
        // Or if it's empty we ignore? For description, an empty string could mean wiping it out. Let's update it anyway.
        // Wait, how do we distinguish empty vs not sent without json.RawMessage? Let's just assign.
		// A cleaner pattern is just updating all provided string pointers, but since it's direct scalar:
	}
    // I'll keep the logic simple, matching Create.
	if req.Description != "" {
		deadline.Description = strings.TrimSpace(req.Description)
	}
	
	if req.DueAt != "" {
		dueAt, err := time.Parse(time.RFC3339, req.DueAt)
		if err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "due_at must use RFC3339 format", nil)
			return
		}
		deadline.DueAt = dueAt.UTC()
	}

	if req.Priority != nil {
		deadline.Priority = *req.Priority
	}

	if err := h.DB.Save(&deadline).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update deadline", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "deadline updated", toDeadlineResponse(deadline, deadline.Matkul.Name))
}

func (h *DeadlineHandler) ToggleStatus(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	deadlineID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid deadline id", nil)
		return
	}

	var deadline models.Deadline
	if err := h.DB.Preload("Matkul").Where("id = ? AND user_id = ?", deadlineID, userID).First(&deadline).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "deadline not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to find deadline", nil)
		return
	}

	if deadline.Status == "todo" {
		deadline.Status = "done"
	} else {
		deadline.Status = "todo"
	}

	if err := h.DB.Save(&deadline).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to toggle status", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "deadline status toggled", toDeadlineResponse(deadline, deadline.Matkul.Name))
}

// Delete menghapus tugas berdasarkan ID, hanya jika tugas milik user yang sedang login
func (h *DeadlineHandler) Delete(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	idParam := c.Param("id")
	if idParam == "" {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "missing deadline id", nil)
		return
	}
	deadlineID, err := uuid.Parse(idParam)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid deadline id", nil)
		return
	}

	var deadline models.Deadline
	if err := h.DB.Where("id = ? AND user_id = ?", deadlineID, userID).First(&deadline).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "deadline not found or not owned by user", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to find deadline", nil)
		return
	}

	if err := h.DB.Delete(&deadline).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete deadline", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "deadline deleted", nil)
}
