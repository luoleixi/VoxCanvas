package router

import (
	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/handler"
)

func Setup() *gin.Engine {
	r := gin.Default()

	r.GET("/", handler.Home)
	r.GET("/health", handler.Health)

	api := r.Group("/api/v1")
	{
		api.POST("/session/start", handler.SessionStart)
		api.POST("/draw/understand", handler.DrawUnderstand)
	}

	return r
}
