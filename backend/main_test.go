package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"voxcanvas/backend/internal/llm"
	"voxcanvas/backend/internal/model"
	"voxcanvas/backend/internal/router"
	"voxcanvas/backend/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testSetup() *gin.Engine {
	sessionSvc := &service.SessionService{}
	drawSvc := &service.DrawService{
		Dev:        service.NewDevStore(),
		Generated:  service.NewGeneratedStore(),
		Sessions:   sessionSvc,
		Classifier: &llm.MockClassifier{},
		Refiner:    &llm.MockRefiner{},
		Generator:  &llm.MockGenerator{},
	}
	return router.Setup(drawSvc, sessionSvc)
}

func TestHealthHandler(t *testing.T) {
	r := testSetup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp model.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Code != 200 {
		t.Fatalf("expected code 200, got %d", resp.Code)
	}
}

func TestSessionStartHandler(t *testing.T) {
	r := testSetup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/start", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp model.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Code != 200 {
		t.Fatalf("expected code 200, got %d", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map")
	}

	sessionID, ok := data["session_id"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("expected non-empty session_id, got %v", data["session_id"])
	}

	if !strings.HasPrefix(sessionID, "sess_") {
		t.Fatalf("expected session_id to start with sess_, got %s", sessionID)
	}

	if cookieValue(rec, "vox_client_id") == "" {
		t.Fatal("expected vox_client_id cookie")
	}

	if got := cookieValue(rec, "vox_session_id"); got != sessionID {
		t.Fatalf("expected vox_session_id cookie %s, got %s", sessionID, got)
	}
}

func TestDrawUnderstandHandler(t *testing.T) {
	r := testSetup()

	body := `{"sentences":"画一只猫"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/draw/understand", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if cookieValue(rec, "vox_client_id") == "" {
		t.Fatal("expected vox_client_id cookie")
	}

	if cookieValue(rec, "vox_session_id") == "" {
		t.Fatal("expected vox_session_id cookie")
	}

	var resp model.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Code != 200 {
		t.Fatalf("expected code 200, got %d", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map")
	}

	if data["op"] == "" {
		t.Fatal("expected non-empty op")
	}

	if data["text"] == "" {
		t.Fatal("expected non-empty text")
	}

	if data["image"] != "" {
		t.Fatalf("expected empty image for requirement, got %v", data["image"])
	}
}

func TestDrawUnderstandSwitchSessionUpdatesCookie(t *testing.T) {
	r := testSetup()

	body := `{"sentences":"切换会话"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/draw/understand", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "vox_client_id", Value: "client_test"})
	req.AddCookie(&http.Cookie{Name: "vox_session_id", Value: "sess_old"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp model.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map")
	}

	if data["op"] != "switch_session" {
		t.Fatalf("expected switch_session op, got %v", data["op"])
	}

	newSessionID := cookieValue(rec, "vox_session_id")
	if newSessionID == "" || newSessionID == "sess_old" {
		t.Fatalf("expected new vox_session_id cookie, got %s", newSessionID)
	}
}

func TestDrawUnderstandUndoReturnsLastGeneratedResult(t *testing.T) {
	r := testSetup()

	cookies := []*http.Cookie{
		{Name: "vox_client_id", Value: "client_test"},
		{Name: "vox_session_id", Value: "sess_test"},
	}

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/draw/understand", strings.NewReader(`{"sentences":"画一只猫"}`))
	req1.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req1.AddCookie(cookie)
	}
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected requirement status 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/draw/understand", strings.NewReader(`{"sentences":"生成图片"}`))
	req2.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected generate status 200, got %d", rec2.Code)
	}

	var genResp model.Response
	if err := json.Unmarshal(rec2.Body.Bytes(), &genResp); err != nil {
		t.Fatalf("failed to unmarshal generate response: %v", err)
	}
	genData := genResp.Data.(map[string]interface{})
	generatedImage, _ := genData["image"].(string)
	if generatedImage == "" {
		t.Fatal("expected generated image")
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/draw/understand", strings.NewReader(`{"sentences":"撤销"}`))
	req3.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req3.AddCookie(cookie)
	}
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected undo status 200, got %d", rec3.Code)
	}

	var undoResp model.Response
	if err := json.Unmarshal(rec3.Body.Bytes(), &undoResp); err != nil {
		t.Fatalf("failed to unmarshal undo response: %v", err)
	}
	undoData := undoResp.Data.(map[string]interface{})
	if undoData["op"] != "undo" {
		t.Fatalf("expected undo op, got %v", undoData["op"])
	}
	if undoData["text"] == "" {
		t.Fatal("expected undo text")
	}
	if undoData["image"] != generatedImage {
		t.Fatal("expected undo to return last generated image")
	}
}

func TestDrawUnderstandHandlerInvalidBody(t *testing.T) {
	r := testSetup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/draw/understand", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func cookieValue(rec *httptest.ResponseRecorder, name string) string {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}
