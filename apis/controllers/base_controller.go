package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/utils"
)

func Healthz(c *gin.Context) {
	utils.RespondOK(c, gin.H{"result": "healthy"})
}
