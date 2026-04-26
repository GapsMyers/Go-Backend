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
	Name     string `json:"name"`
	Title    string `json:"title"`
	MatkulID string `json:"matkul_id" binding:"required"`
	Deadline string `json:"deadline" binding:"required"`
	Priority string `json:"priority"`
}

type updateTaskRequest struct {
	IsDone *bool `json:"is_done" binding:"required"`
}

type taskResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	IsDone     bool   `json:"is_done"`
	Deadline   string `json:"deadline"`
	Priority   string `json:"priority"`
	MatkulID   string `json:"matkul_id"`
	MatkulName string `json:"matkul_name"`
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

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(req.Name)
	}
	if title == "" {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "title or name is required", nil)
		return
	}

	priority := normalizeTaskPriority(req.Priority)
	if !isValidTaskPriority(priority) {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "priority must be one of: low, medium, high", nil)
		return
	}

	matkulID, err := uuid.Parse(req.MatkulID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid matkul_id", nil)
		return
	}

	deadline, err := parseTaskDeadline(req.Deadline)
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
		Title:    title,
		MatkulID: matkulID,
		Priority: priority,
		Deadline: deadline,
		IsDone:   false,
	}

	if err := h.DB.Create(&task).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create task", nil)
		return
	}

	task.Matkul = matkul

	writeSuccess(c, http.StatusCreated, "task created", toTaskResponse(task))
}

func (h *TaskHandler) List(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	matkulParam := strings.TrimSpace(c.Query("matkul_id"))
	query := h.DB.
		Preload("Matkul").
		Joins("JOIN matkuls ON matkuls.id = tasks.matkul_id").
		Where("matkuls.user_id = ?", userID).
		Order("tasks.deadline ASC")

	if matkulParam != "" {
		matkulID, err := uuid.Parse(matkulParam)
		if err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid matkul_id", nil)
			return
		}

		// Keep ownership check explicit for clearer errors when matkul does not exist.
		var matkul models.Matkul
		if err := h.DB.Select("id").Where("id = ? AND user_id = ?", matkulID, userID).First(&matkul).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeError(c, http.StatusNotFound, "NOT_FOUND", "matkul not found", nil)
				return
			}
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
			return
		}

		query = query.Where("tasks.matkul_id = ?", matkulID)
	}

	var tasks []models.Task
	if err := query.Find(&tasks).Error; err != nil {
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
	if err := h.DB.
		Preload("Matkul").
		Joins("JOIN matkuls ON matkuls.id = tasks.matkul_id").
		Where("tasks.id = ? AND matkuls.user_id = ?", taskID, userID).
		First(&task).Error; err != nil {
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

	if err := h.DB.Preload("Matkul").First(&task, "id = ?", task.ID).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch updated task", nil)
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
	if err := h.DB.
		Joins("JOIN matkuls ON matkuls.id = tasks.matkul_id").
		Where("tasks.id = ? AND matkuls.user_id = ?", taskID, userID).
		First(&task).Error; err != nil {
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
		ID:         t.ID.String(),
		Title:      t.Title,
		IsDone:     t.IsDone,
		Deadline:   t.Deadline.UTC().Format("2006-01-02T15:04:05.000Z"),
		Priority:   t.Priority,
		MatkulID:   t.MatkulID.String(),
		MatkulName: t.Matkul.Name,
	}
}

func normalizeTaskPriority(priority string) string {
	p := strings.ToLower(strings.TrimSpace(priority))
	if p == "" {
		return "medium"
	}
	return p
}

func isValidTaskPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

func parseTaskDeadline(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, errors.New("deadline is required")
	}

	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, errors.New("invalid deadline format")
}
