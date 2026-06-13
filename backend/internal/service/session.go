package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"voxcanvas/backend/internal/db"
)

type SessionService struct {
	DB *db.DB
}

type SessionSummary struct {
	SessionID string
	Title     string
	Summary   string
	Dev       string
	UpdatedAt string
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

func (s *SessionService) SetDev(sessionID, dev string) error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.UpdateSessionDev(sessionID, dev)
}

func (s *SessionService) List(clientID string, limit int) ([]SessionSummary, error) {
	if s == nil || s.DB == nil {
		return []SessionSummary{}, nil
	}
	rows, err := s.DB.ListSessionsByClient(clientID, limit)
	if err != nil {
		return nil, err
	}
	sessions := make([]SessionSummary, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, SessionSummary{
			SessionID: row.SessionID,
			Title:     row.Title,
			Summary:   row.Summary,
			Dev:       row.Dev,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return sessions, nil
}

func BuildSessionMeta(text string) (string, string) {
	summary := normalizeSessionText(text)
	if summary == "" {
		return "", ""
	}
	return truncateRunes(summary, 18), truncateRunes(summary, 80)
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

func normalizeSessionText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}
