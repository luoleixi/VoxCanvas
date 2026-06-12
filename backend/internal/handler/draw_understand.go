package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/model"
)

func DrawUnderstand(c *gin.Context) {
	var req model.DrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code: 400,
			Msg:  "invalid request body",
			Data: nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: model.DrawData{
			Op:      "requirement",
			Content: req.Sentences,
		},
	})
}
