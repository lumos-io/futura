package routes

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/controllers"
	"github.com/opisvigilant/futura/apis/middleware"
)

func SetupRouter(embeddedFiles embed.FS) (*gin.Engine, error) {
	router := gin.Default()

	router.Use(middleware.TraceIDMiddleware())

	// Auth routes
	auth := router.Group("/auth")
	{
		auth.GET("/google/login", controllers.GoogleLogin)
		auth.GET("/google/callback", controllers.GoogleCallback)
		auth.GET("/github/login", controllers.GithubLogin)
		auth.GET("/github/callback", controllers.GithubCallback)
		auth.GET("/me", middleware.AuthMiddleware(), controllers.MeHandler)
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

	// Frontend serving
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	distDir := fmt.Sprintf("%s/public/", dir)
	viteStaticFS := os.DirFS(distDir)

	// ref: https://github.com/gin-gonic/gin/issues/3709
	router.Use(static.Serve("/", static.LocalFile(distDir, true)))
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.RequestURI, "/assets") {
			c.FileFromFS(c.Request.URL.Path, http.FS(viteStaticFS))
			return
		}
		if !strings.HasPrefix(c.Request.RequestURI, "/api") {
			c.FileFromFS("", http.FS(viteStaticFS))
			return
		}
	})

	return router, nil
}
