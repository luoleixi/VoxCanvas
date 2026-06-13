package router

import (
	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/handler"
	"voxcanvas/backend/internal/service"
)

func Setup(drawSvc *service.DrawService, sessionSvc *service.SessionService) *gin.Engine {
	r := gin.Default()

	r.GET("/", handler.Home)
	r.GET("/health", handler.Health)

	drawH := &handler.DrawHandler{Svc: drawSvc}
	sessionH := &handler.SessionHandler{Sessions: sessionSvc}

	api := r.Group("/api/v1")
	{
		api.POST("/session/start", sessionH.Start)
		api.POST("/draw/understand", drawH.Understand)
	}

	return r
}
