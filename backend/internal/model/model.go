package model

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

type DrawData struct {
	Op        string `json:"op"`
	Text      string `json:"text"`
	Image     string `json:"image"`
	SessionID string `json:"-"`
}
