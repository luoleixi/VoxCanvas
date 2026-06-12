package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/model"
)

func SessionStart(c *gin.Context) {
	now := time.Now()
	sessionID := fmt.Sprintf("sess_%s_%s%03d", now.Format("20060102"), now.Format("150405"), now.UnixMilli()%1000)
	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: model.SessionData{
			SessionID: sessionID,
		},
	})
}
