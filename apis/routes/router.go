package routes

import "github.com/gin-gonic/gin"

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Auth routes
	auth := router.Group("/auth")
	{
		auth.GET("/google/login", controllers.GoogleLogin)
		auth.GET("/google/callback", controllers.GoogleCallback)
		auth.GET("/github/login", controllers.GithubLogin)
		auth.GET("/github/callback", controllers.GithubCallback)
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

			orgUsers := org.Group("/:org_id/users")
			{
				orgUsers.GET("/", controllers.GetUsers)
				orgUsers.POST("/", controllers.CreateUser)
				orgUsers.PUT("/:user_id", controllers.UpdateUser)
				orgUsers.DELETE("/:user_id", controllers.DeleteUser)
			}
		}
	}

	return router
}
