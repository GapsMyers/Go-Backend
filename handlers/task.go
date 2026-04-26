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

type TaskHandler struct {
	DB *gorm.DB
}

func NewTaskHandler(db *gorm.DB) *TaskHandler {
	return &TaskHandler{DB: db}
}

type createTaskRequest struct {
	Title    string `json:"title" binding:"required,min=1"`
	MatkulID string `json:"matkul_id" binding:"required,uuid"`
	Deadline string `json:"deadline" binding:"required"` // Assuming format 2006-01-02
}

type updateTaskRequest struct {
	IsDone *bool `json:"is_done" binding:"required"`
}

type taskResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	MatkulID  string `json:"matkul_id"`
	IsDone    bool   `json:"is_done"`
	Deadline  string `json:"deadline"`
	CreatedAt string `json:"created_at"`
}

func (h *TaskHandler) Create(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", err.Error())
		return
	}

	matkulID, err := uuid.Parse(req.MatkulID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid matkul_id", nil)
		return
	}

	// Format: "YYYY-MM-DD" or RFC3339, prompt says "2026-04-26" array length is 10
	var deadline time.Time
	if len(req.Deadline) == 10 {
		deadline, err = time.Parse("2006-01-02", req.Deadline)
	} else {
		deadline, err = time.Parse(time.RFC3339, req.Deadline)
	}

	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid deadline format", nil)
		return
	}

	// Verify matkul ownership
	var matkul models.Matkul
	if err := h.DB.Where("id = ? AND user_id = ?", matkulID, userID).First(&matkul).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "matkul not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch matkul", nil)
		return
	}

	task := models.Task{
		Title:    req.Title,
		MatkulID: matkulID,
		Deadline: deadline,
		IsDone:   false,
	}

	if err := h.DB.Create(&task).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create task", nil)
		return
	}

	writeSuccess(c, http.StatusCreated, "task created", toTaskResponse(task))
}

func (h *TaskHandler) List(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	matkulParam := strings.TrimSpace(c.Query("matkul_id"))
	if matkulParam == "" {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "matkul_id query is required", nil)
		return
	}

	matkulID, err := uuid.Parse(matkulParam)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid matkul_id", nil)
		return
	}

	// Output tasks for this specific matkul, but first ensure the user owns this matkul
	var matkul models.Matkul
	if err := h.DB.Select("id").Where("id = ? AND user_id = ?", matkulID, userID).First(&matkul).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "matkul not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
		return
	}

	var tasks []models.Task
	if err := h.DB.Where("matkul_id = ?", matkulID).Order("deadline ASC").Find(&tasks).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tasks", nil)
		return
	}

	response := make([]taskResponse, 0, len(tasks))
	for _, task := range tasks {
		response = append(response, toTaskResponse(task))
	}

	writeSuccess(c, http.StatusOK, "tasks fetched", response)
}

func (h *TaskHandler) Update(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth", nil)
		return
	}

	taskID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid task id", nil)
		return
	}

	var task models.Task
	// Perform a join or preload to check ownership
	if err := h.DB.Joins("Matkul").Where("tasks.id = ? AND Matkul.user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "task not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "query failed", nil)
		return
	}

	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", err.Error())
		return
	}

	if req.IsDone != nil {
		task.IsDone = *req.IsDone
	}

	if err := h.DB.Save(&task).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save task", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "task updated", toTaskResponse(task))
}

func (h *TaskHandler) Delete(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth", nil)
		return
	}

	taskID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid task id", nil)
		return
	}

	var task models.Task
	if err := h.DB.Joins("Matkul").Where("tasks.id = ? AND Matkul.user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "task not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "query failed", nil)
		return
	}

	if err := h.DB.Delete(&task).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete task", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "task deleted", nil)
}

func toTaskResponse(t models.Task) taskResponse {
	return taskResponse{
		ID:        t.ID.String(),
		Title:     t.Title,
		MatkulID:  t.MatkulID.String(),
		IsDone:    t.IsDone,
		Deadline:  t.Deadline.Format("2006-01-02"),
		CreatedAt: t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
