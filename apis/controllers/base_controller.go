package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/utils"
	"github.com/opisvigilant/futura/apis/version"
)

func Healthz(c *gin.Context) {
	utils.RespondOK(c, gin.H{"result": "healthy"})
}

func Version(c *gin.Context) {
	c.JSON(200, gin.H{
		"version":    version.Version,
		"commit_sha": version.CommitSHA,
		"build_time": version.BuildTime,
	})
}
