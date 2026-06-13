package handler

import (
	"log"
	"net/http"
	"time"

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

	start := time.Now()
	log.Printf("[DRAW] understand start client_id=%s session_id=%s sentence_len=%d", clientID, sessionID, len(req.Sentences))
	data, err := h.Svc.Handle(clientID, sessionID, req.Sentences)
	if err != nil {
		log.Printf("[DRAW] understand error client_id=%s session_id=%s err=%v duration_ms=%d", clientID, sessionID, err, time.Since(start).Milliseconds())
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
	log.Printf("[DRAW] understand done client_id=%s session_id=%s op=%s text_len=%d image_len=%d duration_ms=%d", clientID, sessionID, data.Op, len(data.Text), len(data.Image), time.Since(start).Milliseconds())

	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	})
}
