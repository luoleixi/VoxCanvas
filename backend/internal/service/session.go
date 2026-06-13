package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"voxcanvas/backend/internal/db"
)

type SessionService struct {
	DB *db.DB
}

func (s *SessionService) Create(clientID, sessionID string) error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.UpsertSession(clientID, sessionID)
}

func (s *SessionService) Touch(clientID, sessionID string) error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.UpsertSession(clientID, sessionID)
}

func NewSessionID() string {
	now := time.Now()
	return fmt.Sprintf("sess_%s_%s_%s", now.Format("20060102"), now.Format("150405"), randomSessionHex(4))
}

func randomSessionHex(size int) string {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
