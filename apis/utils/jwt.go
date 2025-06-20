package utils

import "github.com/gin-gonic/gin"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// token := c.GetHeader("Authorization")
		// validate JWT and set user in context
		c.Next()
	}
}
