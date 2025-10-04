package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalPages int `json:"totalPages"`
	TotalItems int `json:"totalItems"`
}

type ApiResponse struct {
	Status     string          `json:"status"`               // "success" or "error"
	Message    string          `json:"message,omitempty"`    // Optional
	Data       any             `json:"data,omitempty"`       // Optional
	Pagination *PaginationMeta `json:"pagination,omitempty"` // Optional
	ErrorCode  string          `json:"errorCode,omitempty"`  // For internal error mapping
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

func RespondError(c *gin.Context, code int, errCode, message string) {
	resp := ApiResponse{
		Status:    "error",
		Message:   message,
		ErrorCode: errCode,
	}
	c.JSON(code, resp)
}
