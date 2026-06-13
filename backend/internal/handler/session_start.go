package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/model"
	"voxcanvas/backend/internal/service"
)

type SessionHandler struct {
	Sessions *service.SessionService
}

func (h *SessionHandler) Start(c *gin.Context) {
	clientID := ensureClientID(c)
	sessionID := newSessionID()
	if h.Sessions != nil {
		if err := h.Sessions.Create(clientID, sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, model.Response{
				Code: 500,
				Msg:  err.Error(),
				Data: nil,
			})
			return
		}
	}
	setSessionID(c, sessionID)
	log.Printf("[SESSION] start client_id=%s session_id=%s", clientID, sessionID)

	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: model.SessionData{
			SessionID: sessionID,
		},
	})
}
