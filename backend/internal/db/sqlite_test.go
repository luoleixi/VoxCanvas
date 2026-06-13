package db

import "testing"

func TestInsertSessionEvent(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(dir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.UpsertSession("client_test", "sess_test"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := database.InsertSessionEvent(SessionEvent{
		SessionID:       "sess_test",
		EventType:       "requirement_refined",
		PreviousImageID: 12,
		Sentence:        "画一只猫",
		Dev:             "一只猫",
		BeforeDev:       "",
		BeforeImageID:   12,
	}); err != nil {
		t.Fatalf("failed to insert session event: %v", err)
	}

	var count int
	if err := database.conn.QueryRow(`
		SELECT COUNT(*)
		FROM session_events
		WHERE session_id = ? AND event_type = ? AND sentence = ? AND dev = ?
	`, "sess_test", "requirement_refined", "画一只猫", "一只猫").Scan(&count); err != nil {
		t.Fatalf("failed to query session event: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
}

func TestRecordSentenceWritesSentenceAndEventInTransaction(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(dir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.UpsertSession("client_test", "sess_test"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	sentenceID, err := database.RecordSentence("sess_test", 0, "画一只猫", "user_input", "")
	if err != nil {
		t.Fatalf("failed to record sentence: %v", err)
	}
	if sentenceID == 0 {
		t.Fatal("expected non-zero sentence id")
	}

	var count int
	if err := database.conn.QueryRow(`
		SELECT COUNT(*)
		FROM session_events
		WHERE session_id = ? AND event_type = ? AND sentence_id = ? AND sentence = ?
	`, "sess_test", "sentence", sentenceID, "画一只猫").Scan(&count); err != nil {
		t.Fatalf("failed to query sentence event: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 sentence event, got %d", count)
	}
}

func TestSessionTitleSummaryAreUpdatedAndListed(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(dir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.UpsertSession("client_test", "sess_test"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := database.RecordRequirementRefined("sess_test", "月光下的猫", "月光下的猫坐在森林里", SessionEvent{
		SessionID: "sess_test",
		EventType: "requirement_refined",
		Dev:       "月光下的猫坐在森林里",
	}); err != nil {
		t.Fatalf("failed to record requirement refined: %v", err)
	}

	sessions, err := database.ListSessionsByClient("client_test", 20)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Title != "月光下的猫" {
		t.Fatalf("expected title to be updated, got %q", sessions[0].Title)
	}
	if sessions[0].Summary != "月光下的猫坐在森林里" {
		t.Fatalf("expected summary to be updated, got %q", sessions[0].Summary)
	}
}
