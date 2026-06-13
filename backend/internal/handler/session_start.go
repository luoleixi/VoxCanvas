package handler

import (
	"log"
	"net/http"
	"strconv"

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

func (h *SessionHandler) List(c *gin.Context) {
	clientID := ensureClientID(c)
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		limit = 20
	}

	if h.Sessions == nil {
		c.JSON(http.StatusOK, model.Response{
			Code: 200,
			Msg:  "success",
			Data: []model.SessionSummaryData{},
		})
		return
	}

	sessions, err := h.Sessions.List(clientID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Code: 500,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}

	data := make([]model.SessionSummaryData, 0, len(sessions))
	for _, session := range sessions {
		data = append(data, model.SessionSummaryData{
			SessionID: session.SessionID,
			Title:     session.Title,
			Summary:   session.Summary,
			Dev:       session.Dev,
			UpdatedAt: session.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	})
}
