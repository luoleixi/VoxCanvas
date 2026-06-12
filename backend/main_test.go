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
	drawSvc := &service.DrawService{
		Dev:        service.NewDevStore(),
		Classifier: &llm.MockClassifier{},
		Refiner:    &llm.MockRefiner{},
		Generator:  &llm.MockGenerator{},
	}
	return router.Setup(drawSvc)
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

	if data["content"] == "" {
		t.Fatal("expected non-empty content")
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
