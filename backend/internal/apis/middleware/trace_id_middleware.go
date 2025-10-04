package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const traceIDKey = "TraceID"

func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
			c.Request.Header.Set("X-Trace-ID", traceID)
		}
		// Store in context
		c.Set(traceIDKey, traceID)

		// Optional: Add to response headers too
		c.Writer.Header().Set("X-Trace-ID", traceID)

		c.Next()
	}
}

func GetTraceID(c *gin.Context) string {
	if tid, exists := c.Get(traceIDKey); exists {
		return tid.(string)
	}
	return ""
}
