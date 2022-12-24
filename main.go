package main

import (
	"gin-leakybucket-middleware/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()
	r.GET("/RL", middleware.SetRequest, middleware.RLimiter("ip", 10, 1),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"msg": "hit bucket"})
		})

	r.Run("localhost:8080")
}
