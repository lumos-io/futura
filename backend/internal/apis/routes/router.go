package routes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/backend/internal/apis/controllers"
	"github.com/opisvigilant/futura/backend/internal/apis/middleware"
	"github.com/opisvigilant/futura/backend/internal/shared/config"
)

func SetupRouter(config *config.Configuration) (*gin.Engine, error) {
	router := gin.Default()

	// Security headers
	router.Use(middleware.SecurityHeadersMiddleware())

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

	router.GET("/healthz", controllers.Healthz)
	router.GET("/version", controllers.Version)

	// Initialize analytics controller
	analyticsCtrl, err := controllers.NewAnalyticsController(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize analytics controller: %w", err)
	}

	// Auth routes
	a := controllers.NewAuthController(config)
	auth := router.Group("/auth")
	{
		auth.GET("/google/login", a.GoogleLogin)
		auth.GET("/google/callback", middleware.OAuthCallbackRateLimiter.Middleware(), a.GoogleCallback)
		auth.GET("/github/login", a.GithubLogin)
		auth.GET("/github/callback", middleware.OAuthCallbackRateLimiter.Middleware(), a.GithubCallback)
		auth.GET("/me", middleware.AuthMiddleware(), a.MeHandler)
		auth.POST("/logout", middleware.AuthMiddleware(), a.Logout)
		auth.POST("/refresh", middleware.RefreshTokenRateLimiter.Middleware(), a.RefreshToken)
	}

	// Protected routes
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware()) // JWT auth
	{
		// organization endpoints
		org := api.Group("/organizations")
		{
			org.GET("/", controllers.GetOrganizations)
			org.POST("/", controllers.CreateOrganization)
			org.GET("/:org_id", controllers.GetOrganization)
			org.PUT("/:org_id", controllers.UpdateOrganization)
			org.DELETE("/:org_id", controllers.DeleteOrganization)

			orgUsers := org.Group("/:org_id/users")
			{
				orgUsers.GET("/", controllers.GetUsers)
				orgUsers.POST("/invite", controllers.InviteUser)
				orgUsers.PUT("/:user_id", controllers.UpdateUser)
				orgUsers.DELETE("/:user_id", controllers.DeleteUser)
			}

			orgTeams := org.Group("/:org_id/teams")
			{
				orgTeams.GET("/", controllers.GetTeams)
				orgTeams.POST("/", controllers.CreateTeam)
				orgTeams.PUT("/:team_id", controllers.UpdateTeam)
				orgTeams.DELETE("/:team_id", controllers.DeleteTeam)
			}

			cc, err := controllers.NewConnectController(config)
			if err != nil {
				return nil, err
			}
			ssec, err := controllers.NewSSEController(config.Redis)
			if err != nil {
				return nil, err
			}
			orgConnects := org.Group("/:org_id/connects")
			{
				orgConnects.GET("/", cc.GetConnects)
				orgConnects.POST("/", cc.CreateConnect)
				orgConnects.DELETE("/:connect_id", cc.DeleteConnect)
				orgConnects.POST("/test-connection", cc.TestConnection)
				// sse endpoint
				orgConnects.GET("/fetch", ssec.FetchClustersResultHandler)
				// TODO: move the DELETE endpoint here to make it a nicer UX
				// orgConnects.GET("/delete", ssec.DeleteClustersResultHandler)
			}

			clusterController, err := controllers.NewClusterController(config)
			if err != nil {
				return nil, err
			}
			orgClusters := orgConnects.Group("/:connect_id/clusters")
			{
				orgClusters.GET("/", clusterController.GetClusters)
				orgClusters.DELETE("/:cluster_id", clusterController.DeleteCluster)
			}

			// Cluster-specific analytics endpoints (direct access by cluster ID)
			analytics := org.Group("/:org_id/clusters")
			{
				analytics.GET("/:cluster_id/services", analyticsCtrl.GetServices)
				analytics.GET("/:cluster_id/overview", analyticsCtrl.GetOverviewMetrics)
				analytics.GET("/:cluster_id/cluster-config", analyticsCtrl.GetClusterConfig)
				analytics.GET("/:cluster_id/metrics", analyticsCtrl.GetMetrics)
				analytics.GET("/:cluster_id/nodes", analyticsCtrl.GetNodes)
				analytics.GET("/:cluster_id/events", analyticsCtrl.GetEvents)
				analytics.GET("/:cluster_id/slo-metrics", analyticsCtrl.GetSLOMetricsSSE)
				analytics.GET("/:cluster_id/nodes/stream", analyticsCtrl.GetNodesSSE)
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
