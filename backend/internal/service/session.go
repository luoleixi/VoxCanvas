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

type SwitchTarget struct {
	Session SessionSummary
	Found   bool
	Reason  string
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

func (s *SessionService) ResolveSwitchTarget(clientID, currentSessionID, sentence string) (*SwitchTarget, error) {
	if isNewSessionRequest(sentence) {
		return &SwitchTarget{Found: false, Reason: "new_session"}, nil
	}
	sessions, err := s.List(clientID, 50)
	if err != nil {
		return nil, err
	}

	others := make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		if session.SessionID != currentSessionID {
			others = append(others, session)
		}
	}
	if len(others) == 0 {
		return &SwitchTarget{Found: false, Reason: "no_history"}, nil
	}

	query := normalizeMatchText(sentence)
	bestScore := 0
	var best SessionSummary
	for _, session := range others {
		score := sessionMatchScore(query, session)
		if score > bestScore {
			bestScore = score
			best = session
		}
	}
	if bestScore > 0 {
		return &SwitchTarget{Session: best, Found: true, Reason: "matched_summary"}, nil
	}

	if isRecentHistoryRequest(sentence) {
		return &SwitchTarget{Session: others[0], Found: true, Reason: "recent_history"}, nil
	}
	return &SwitchTarget{Found: false, Reason: "no_match"}, nil
}

func BuildSessionMeta(text string) (string, string) {
	summary := normalizeSessionText(text)
	if summary == "" {
		return "", ""
	}
	return truncateRunes(summary, 18), truncateRunes(summary, 80)
}

func isNewSessionRequest(sentence string) bool {
	s := normalizeMatchText(sentence)
	keywords := []string{"新建会话", "新会话", "新的会话", "另一个会话", "切换会话", "换个会话", "下一个会话", "重新开"}
	for _, keyword := range keywords {
		if strings.Contains(s, normalizeMatchText(keyword)) {
			return true
		}
	}
	return false
}

func isRecentHistoryRequest(sentence string) bool {
	s := normalizeMatchText(sentence)
	keywords := []string{"上一个", "上次", "刚才", "之前", "最近", "历史", "切回", "返回", "回到"}
	for _, keyword := range keywords {
		if strings.Contains(s, normalizeMatchText(keyword)) {
			return true
		}
	}
	return false
}

func sessionMatchScore(query string, session SessionSummary) int {
	score := 0
	title := normalizeMatchText(session.Title)
	summary := normalizeMatchText(session.Summary)
	dev := normalizeMatchText(session.Dev)
	if title != "" && strings.Contains(query, title) {
		score += 100 + len([]rune(title))
	}
	if summary != "" && strings.Contains(query, summary) {
		score += 60 + len([]rune(summary))
	}
	if dev != "" && strings.Contains(query, dev) {
		score += 30 + len([]rune(dev))
	}

	for _, token := range extractMatchTokens(query) {
		if len([]rune(token)) < 2 {
			continue
		}
		if title != "" && strings.Contains(title, token) {
			score += 20 + len([]rune(token))
		}
		if summary != "" && strings.Contains(summary, token) {
			score += 10 + len([]rune(token))
		}
		if dev != "" && strings.Contains(dev, token) {
			score += 5 + len([]rune(token))
		}
	}
	return score
}

func extractMatchTokens(query string) []string {
	stopWords := []string{
		"打开", "切回", "切换", "回到", "返回", "历史", "会话", "那张", "那个", "这个", "作品", "画", "最近", "上一个", "上次", "刚才", "之前",
	}
	cleaned := query
	for _, stopWord := range stopWords {
		cleaned = strings.ReplaceAll(cleaned, normalizeMatchText(stopWord), " ")
	}
	return strings.Fields(cleaned)
}

func normalizeMatchText(text string) string {
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "", "，", "", "。", "", "！", "", "？", "", ",", "", ".", "", "!", "", "?", "", "“", "", "”", "", "\"", "")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(text)))
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
