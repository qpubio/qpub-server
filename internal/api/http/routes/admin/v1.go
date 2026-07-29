package admin

import "github.com/gin-gonic/gin"

func SetupRoutes(router *gin.Engine, services interface{}, logger interface{}) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// admin := router.Group("/admin")

	// v1 := admin.Group("/v1")
	// {

	// }
}
