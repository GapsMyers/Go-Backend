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
	DueAt       string `json:"due_at" binding:"required,datetime=2006-01-02T15:04:05Z07:00"`
}

type deadlineResponse struct {
	ID          string `json:"id"`
	MatkulID    string `json:"matkul_id"`
	MatkulName  string `json:"matkul_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueAt       string `json:"due_at"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
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
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		DueAt:       dueAt.UTC(),
		Status:      "todo",
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
		DueAt:       deadline.DueAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Status:      deadline.Status,
		CreatedAt:   deadline.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   deadline.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
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
