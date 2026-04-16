package handlers

import "github.com/gin-gonic/gin"

// SuccessResponse is the standard success payload shape.
type SuccessResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
}

// APIError contains a normalized error payload.
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ErrorResponse is the standard error payload shape.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

func writeSuccess(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, SuccessResponse{
		Data:    data,
		Message: message,
	})
}

func writeError(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
