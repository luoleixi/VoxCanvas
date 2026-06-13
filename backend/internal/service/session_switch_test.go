package service

import (
	"testing"

	"voxcanvas/backend/internal/db"
	"voxcanvas/backend/internal/llm"
)

type fixedSwitchClassifier struct{}

func (f fixedSwitchClassifier) Classify(sentence string) (*llm.IntentResult, error) {
	return &llm.IntentResult{Op: "switch_session", Text: "", Image: ""}, nil
}

func TestResolveSwitchTargetMatchesSessionMetadata(t *testing.T) {
	database, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	sessions := &SessionService{DB: database}
	if err := sessions.Create("client_test", "sess_current"); err != nil {
		t.Fatalf("failed to create current session: %v", err)
	}
	if err := sessions.Create("client_test", "sess_seaside"); err != nil {
		t.Fatalf("failed to create target session: %v", err)
	}
	if err := database.UpdateSessionMeta("sess_seaside", "海边小屋", "夕阳下的海边小屋，天空有几只海鸥"); err != nil {
		t.Fatalf("failed to update target metadata: %v", err)
	}

	target, err := sessions.ResolveSwitchTarget("client_test", "sess_current", "打开海边小屋那张")
	if err != nil {
		t.Fatalf("failed to resolve switch target: %v", err)
	}
	if target == nil || !target.Found {
		t.Fatalf("expected target to be found, got %#v", target)
	}
	if target.Session.SessionID != "sess_seaside" {
		t.Fatalf("expected sess_seaside, got %s", target.Session.SessionID)
	}
}

func TestDrawServiceSwitchesToHistoricalSession(t *testing.T) {
	database, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	sessions := &SessionService{DB: database}
	if err := sessions.Create("client_test", "sess_current"); err != nil {
		t.Fatalf("failed to create current session: %v", err)
	}
	if err := sessions.Create("client_test", "sess_seaside"); err != nil {
		t.Fatalf("failed to create target session: %v", err)
	}
	if err := database.RecordRequirementRefined("sess_seaside", "海边小屋", "夕阳下的海边小屋", db.SessionEvent{
		SessionID: "sess_seaside",
		EventType: "requirement_refined",
		Dev:       "夕阳下的海边小屋",
	}); err != nil {
		t.Fatalf("failed to prepare target session: %v", err)
	}

	draw := &DrawService{
		Dev:        NewDevStore(),
		Generated:  NewGeneratedStore(),
		Sessions:   sessions,
		Classifier: fixedSwitchClassifier{},
		Refiner:    &llm.MockRefiner{},
		Generator:  &llm.MockGenerator{},
		DB:         database,
	}

	data, err := draw.Handle("client_test", "sess_current", "回到海边小屋")
	if err != nil {
		t.Fatalf("failed to handle switch: %v", err)
	}
	if data.Op != "switch_session" {
		t.Fatalf("expected switch_session op, got %s", data.Op)
	}
	if data.SessionID != "sess_seaside" {
		t.Fatalf("expected switch to historical session, got %s", data.SessionID)
	}
	if got := draw.Dev.Get("sess_seaside"); got != "夕阳下的海边小屋" {
		t.Fatalf("expected target dev to be loaded, got %q", got)
	}
}
