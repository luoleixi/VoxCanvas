package model

import "encoding/json"

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type SessionData struct {
	SessionID string `json:"session_id"`
}

type SessionSummaryData struct {
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Dev       string `json:"dev"`
	UpdatedAt string `json:"updated_at"`
}

type DrawRequest struct {
	Sentences string `json:"sentences"`
}

type DrawSessionData struct {
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
}

type DrawData struct {
	Op        string            `json:"op"`
	Text      string            `json:"text"`
	Image     string            `json:"image"`
	Sessions  []DrawSessionData `json:"sessions"`
	SessionID string            `json:"-"`
}

func (d DrawData) MarshalJSON() ([]byte, error) {
	type drawDataJSON struct {
		Op       string            `json:"op"`
		Text     string            `json:"text"`
		Image    string            `json:"image"`
		Sessions []DrawSessionData `json:"sessions"`
	}
	sessions := d.Sessions
	if sessions == nil {
		sessions = []DrawSessionData{}
	}
	return json.Marshal(drawDataJSON{
		Op:       d.Op,
		Text:     d.Text,
		Image:    d.Image,
		Sessions: sessions,
	})
}
