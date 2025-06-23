package routes

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/controllers"
	"github.com/opisvigilant/futura/apis/middleware"
)

func SetupRouter(embeddedFiles embed.FS) (*gin.Engine, error) {
	router := gin.Default()

	// observability
	router.Use(middleware.TraceIDMiddleware())

	// ref: https://github.com/gin-gonic/gin/issues/3709
	// Frontend serving
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	distDir := fmt.Sprintf("%s/public/", dir)
	// viteStaticFS := os.DirFS(distDir)
	router.Use(static.Serve("/", static.LocalFile(distDir, false)))

	// Auth routes
	auth := router.Group("/auth")
	{
		auth.GET("/google/login", controllers.GoogleLogin)
		auth.GET("/google/callback", controllers.GoogleCallback)
		auth.GET("/github/login", controllers.GithubLogin)
		auth.GET("/github/callback", controllers.GithubCallback)
		auth.GET("/me", middleware.AuthMiddleware(), controllers.MeHandler)
		auth.POST("/logout", middleware.AuthMiddleware(), controllers.Logout)
		auth.POST("/refresh", controllers.RefreshToken)
	}

	// Protected routes
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware()) // JWT auth
	{
		org := api.Group("/organizations")
		{
			org.GET("/", controllers.GetOrganizations)
			org.POST("/", controllers.CreateOrganization)
			org.GET("/:id", controllers.GetOrganization)
			org.PUT("/:id", controllers.UpdateOrganization)
			org.DELETE("/:id", controllers.DeleteOrganization)

			orgUsers := org.Group("/:id/users")
			{
				orgUsers.GET("/", controllers.GetUsers)
				orgUsers.POST("/", controllers.CreateUser)
				orgUsers.PUT("/:user_id", controllers.UpdateUser)
				orgUsers.DELETE("/:user_id", controllers.DeleteUser)
			}
		}
	}

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// API routes: do nothing
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/auth") {
			c.Next()
			return
		}
		// Static assets (e.g. /assets/index.js)
		if strings.HasPrefix(path, "/assets") {
			c.File(filepath.Join(distDir, path))
			return
		}
		// For all other routes (e.g. /dashboard), serve index.html
		c.File(filepath.Join(distDir, "index.html"))
	})

	return router, nil
}
