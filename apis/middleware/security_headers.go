package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds security-related HTTP headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS filter in browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Enforce HTTPS in production (HSTS)
		// Note: Only enable this in production with valid HTTPS
		// Uncomment when ready for production:
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Content Security Policy (CSP)
		// This is a basic policy - adjust based on your needs
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none';"
		c.Header("Content-Security-Policy", csp)

		// Permissions Policy (formerly Feature-Policy)
		// Restrict access to browser features
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		c.Next()
	}
}
