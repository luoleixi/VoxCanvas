package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/model"
	"voxcanvas/backend/internal/service"
)

type DrawHandler struct {
	Svc *service.DrawService
}

func (h *DrawHandler) Understand(c *gin.Context) {
	var req model.DrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{
			Code: 400,
			Msg:  "invalid request body",
			Data: nil,
		})
		return
	}

	clientID := ensureClientID(c)
	sessionID := currentSessionID(c)
	if sessionID == "" {
		sessionID = newSessionID()
		setSessionID(c, sessionID)
	}

	data, err := h.Svc.Handle(clientID, sessionID, req.Sentences)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Code: 500,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	if data.Op == "switch_session" {
		setSessionID(c, data.SessionID)
	}

	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	})
}
