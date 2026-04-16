package handlers

import (
	"backend/middleware"
	"backend/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
