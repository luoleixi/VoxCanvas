package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/model"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: gin.H{
			"service": "voxcanvas-backend",
			"status":  "ok",
		},
	})
}
