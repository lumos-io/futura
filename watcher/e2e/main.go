package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.POST("/events", func(c *gin.Context) {
		b, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// TODO: debug only
		fmt.Println(string(b))

		c.JSON(http.StatusOK, gin.H{
			"message": "event accepted",
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080
}
