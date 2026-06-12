package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/model"
)

func SessionStart(c *gin.Context) {
	sessionID := fmt.Sprintf("sess_%s_%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
	c.JSON(http.StatusOK, model.Response{
		Code: 200,
		Msg:  "success",
		Data: model.SessionData{
			SessionID: sessionID,
		},
	})
}
