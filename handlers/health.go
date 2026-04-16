package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	writeSuccess(c, http.StatusOK, "ok", gin.H{"status": "ok"})
}
