package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/service"
)

const (
	clientCookieName  = "vox_client_id"
	sessionCookieName = "vox_session_id"
	cookieMaxAge      = 60 * 60 * 24 * 365
)

func ensureClientID(c *gin.Context) string {
	if clientID, err := c.Cookie(clientCookieName); err == nil && clientID != "" {
		return clientID
	}

	clientID := "client_" + randomHex(16)
	setCookie(c, clientCookieName, clientID)
	return clientID
}

func currentSessionID(c *gin.Context) string {
	sessionID, err := c.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return sessionID
}

func setSessionID(c *gin.Context, sessionID string) {
	setCookie(c, sessionCookieName, sessionID)
}

func newSessionID() string {
	return service.NewSessionID()
}

func setCookie(c *gin.Context, name, value string) {
	c.SetCookie(name, value, cookieMaxAge, "/", "", false, true)
}

func randomHex(size int) string {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
