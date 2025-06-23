package utils

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalPages int `json:"totalPages"`
	TotalItems int `json:"totalItems"`
}

type DebugInfo struct {
	Stack string `json:"stack,omitempty"`
	SQL   string `json:"sql,omitempty"`
}

type ApiResponse struct {
	Status     string          `json:"status"`               // "success" or "error"
	Message    string          `json:"message,omitempty"`    // Optional
	Data       any             `json:"data,omitempty"`       // Optional
	Pagination *PaginationMeta `json:"pagination,omitempty"` // Optional
	ErrorCode  string          `json:"errorCode,omitempty"`  // For internal error mapping
	Debug      *DebugInfo      `json:"debug,omitempty"`      // Only in dev
}

func isDev() bool {
	return os.Getenv("APP_ENV") == "development"
}

func RespondOK(c *gin.Context, data any) {
	resp := ApiResponse{
		Status: "success",
		Data:   data,
	}
	c.JSON(http.StatusOK, resp)
}

func RespondCreated(c *gin.Context, data any) {
	resp := ApiResponse{
		Status: "success",
		Data:   data,
	}
	c.JSON(http.StatusCreated, resp)
}

func RespondWithPagination(c *gin.Context, data any, pagination PaginationMeta) {
	resp := ApiResponse{
		Status:     "success",
		Data:       data,
		Pagination: &pagination,
	}
	c.JSON(http.StatusOK, resp)
}

func RespondError(c *gin.Context, code int, errCode, message string, debug *DebugInfo) {
	resp := ApiResponse{
		Status:    "error",
		Message:   message,
		ErrorCode: errCode,
	}
	if isDev() && debug != nil {
		resp.Debug = debug
	}
	c.JSON(code, resp)
}
