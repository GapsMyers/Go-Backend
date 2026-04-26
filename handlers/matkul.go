package handlers

import (
	"backend/middleware"
	"backend/models"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatkulHandler struct {
	DB *gorm.DB
}

func NewMatkulHandler(db *gorm.DB) *MatkulHandler {
	return &MatkulHandler{DB: db}
}

type createMatkulRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=120"`
	Code     string `json:"code" binding:"max=30"`
	Semester string `json:"semester" binding:"max=20"`
}

type updateMatkulRequest struct {
	Name     string `json:"name" binding:"omitempty,min=2,max=120"`
	Code     string `json:"code" binding:"max=30"`
	Semester string `json:"semester" binding:"max=20"`
}

type matkulResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Semester  string `json:"semester"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (h *MatkulHandler) Create(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	var req createMatkulRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid matkul payload", err.Error())
		return
	}

	matkul := models.Matkul{
		UserID:   userID,
		Name:     strings.TrimSpace(req.Name),
		Code:     strings.TrimSpace(req.Code),
		Semester: strings.TrimSpace(req.Semester),
	}

	if err := h.DB.Create(&matkul).Error; err != nil {
		if isDuplicateError(err) {
			writeError(c, http.StatusConflict, "MATKUL_EXISTS", "matkul already exists for this user", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create matkul", nil)
		return
	}

	writeSuccess(c, http.StatusCreated, "matkul created", toMatkulResponse(matkul))
}

func (h *MatkulHandler) List(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	var matkuls []models.Matkul
	if err := h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&matkuls).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list matkul", nil)
		return
	}

	response := make([]matkulResponse, 0, len(matkuls))
	for _, matkul := range matkuls {
		response = append(response, toMatkulResponse(matkul))
	}

	writeSuccess(c, http.StatusOK, "matkul list fetched", response)
}

func (h *MatkulHandler) Update(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	matkulID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid matkul id", nil)
		return
	}

	var matkul models.Matkul
	if err := h.DB.Where("id = ? AND user_id = ?", matkulID, userID).First(&matkul).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "matkul not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to find matkul", nil)
		return
	}

	var req updateMatkulRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid matkul update payload", err.Error())
		return
	}

	updated := false
	if name := strings.TrimSpace(req.Name); name != "" {
		matkul.Name = name
		updated = true
	}
	if code := strings.TrimSpace(req.Code); code != "" {
		matkul.Code = code
		updated = true
	}
	if semester := strings.TrimSpace(req.Semester); semester != "" {
		matkul.Semester = semester
		updated = true
	}

	if updated {
		if err := h.DB.Save(&matkul).Error; err != nil {
			if isDuplicateError(err) {
				writeError(c, http.StatusConflict, "MATKUL_EXISTS", "matkul already exists for this user", nil)
				return
			}
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update matkul", nil)
			return
		}
	}

	writeSuccess(c, http.StatusOK, "matkul updated", toMatkulResponse(matkul))
}

func toMatkulResponse(matkul models.Matkul) matkulResponse {
	return matkulResponse{
		ID:        matkul.ID.String(),
		Name:      matkul.Name,
		Code:      matkul.Code,
		Semester:  matkul.Semester,
		CreatedAt: matkul.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: matkul.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func tomatkulUpdateResponse(matkul models.Matkul) matkulResponse {
	return matkulResponse{
		ID:        matkul.ID.String(),
		Name:      matkul.Name,
		Code:      matkul.Code,
		Semester:  matkul.Semester,
		CreatedAt: matkul.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: matkul.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
