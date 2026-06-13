package db

import (
	"database/sql"
	"testing"
)

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

func TestRecordUndoToPreviousImageCanUndoRepeatedly(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(dir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.UpsertSession("client_test", "sess_undo"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	firstID, err := database.RecordGeneratedImage("sess_undo", "prompt 1", "image 1", "title 1", "summary 1", SessionEvent{
		SessionID: "sess_undo",
		EventType: "image_generated",
		Dev:       "prompt 1",
	})
	if err != nil {
		t.Fatalf("failed to record first image: %v", err)
	}
	secondID, err := database.RecordGeneratedImage("sess_undo", "prompt 2", "image 2", "title 2", "summary 2", SessionEvent{
		SessionID: "sess_undo",
		EventType: "image_generated",
		Dev:       "prompt 2",
	})
	if err != nil {
		t.Fatalf("failed to record second image: %v", err)
	}
	thirdID, err := database.RecordGeneratedImage("sess_undo", "prompt 3", "image 3", "title 3", "summary 3", SessionEvent{
		SessionID: "sess_undo",
		EventType: "image_generated",
		Dev:       "prompt 3",
	})
	if err != nil {
		t.Fatalf("failed to record third image: %v", err)
	}

	currentID, err := database.CurrentImageID("sess_undo")
	if err != nil {
		t.Fatalf("failed to query current image: %v", err)
	}
	if currentID != thirdID {
		t.Fatalf("expected current image %d, got %d", thirdID, currentID)
	}

	image, err := database.RecordUndoToPreviousImage("sess_undo", SessionEvent{
		SessionID:       "sess_undo",
		EventType:       "undo",
		PreviousImageID: thirdID,
	})
	if err != nil {
		t.Fatalf("failed to undo to third image: %v", err)
	}
	if image == nil || image.ImageID != thirdID || image.Prompt != "prompt 3" || image.Base64Data != "image 3" {
		t.Fatalf("expected third image, got %#v", image)
	}

	image, err = database.RecordUndoToPreviousImage("sess_undo", SessionEvent{
		SessionID:       "sess_undo",
		EventType:       "undo",
		PreviousImageID: thirdID,
	})
	if err != nil {
		t.Fatalf("failed to undo to second image: %v", err)
	}
	if image == nil || image.ImageID != secondID || image.Prompt != "prompt 2" || image.Base64Data != "image 2" {
		t.Fatalf("expected second image, got %#v", image)
	}

	image, err = database.RecordUndoToPreviousImage("sess_undo", SessionEvent{
		SessionID:       "sess_undo",
		EventType:       "undo",
		PreviousImageID: secondID,
	})
	if err != nil {
		t.Fatalf("failed to undo to first image: %v", err)
	}
	if image == nil || image.ImageID != firstID || image.Prompt != "prompt 1" || image.Base64Data != "image 1" {
		t.Fatalf("expected first image, got %#v", image)
	}

	image, err = database.RecordUndoToPreviousImage("sess_undo", SessionEvent{
		SessionID:       "sess_undo",
		EventType:       "undo",
		PreviousImageID: firstID,
	})
	if err != nil {
		t.Fatalf("failed to undo with no previous image: %v", err)
	}
	if image != nil {
		t.Fatalf("expected no previous image, got %#v", image)
	}
}

func TestRecordClearCanBeRestoredByUndo(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(dir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.UpsertSession("client_test", "sess_clear"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	firstID, err := database.RecordGeneratedImage("sess_clear", "prompt 1", "image 1", "title 1", "summary 1", SessionEvent{
		SessionID: "sess_clear",
		EventType: "image_generated",
		Dev:       "prompt 1",
	})
	if err != nil {
		t.Fatalf("failed to record first image: %v", err)
	}
	secondID, err := database.RecordGeneratedImage("sess_clear", "prompt 2", "image 2", "title 2", "summary 2", SessionEvent{
		SessionID: "sess_clear",
		EventType: "image_generated",
		Dev:       "prompt 2",
	})
	if err != nil {
		t.Fatalf("failed to record second image: %v", err)
	}

	if err := database.RecordClear("sess_clear", SessionEvent{
		SessionID:       "sess_clear",
		EventType:       "clear",
		PreviousImageID: secondID,
		BeforeImageID:   secondID,
	}); err != nil {
		t.Fatalf("failed to record clear: %v", err)
	}

	currentID, err := database.CurrentImageID("sess_clear")
	if err != nil {
		t.Fatalf("failed to query current image after clear: %v", err)
	}
	if currentID != 0 {
		t.Fatalf("expected no current image after clear, got %d", currentID)
	}

	image, err := database.RecordUndoToPreviousImage("sess_clear", SessionEvent{
		SessionID:       "sess_clear",
		EventType:       "undo",
		PreviousImageID: 0,
	})
	if err != nil {
		t.Fatalf("failed to undo clear: %v", err)
	}
	if image == nil || image.ImageID != secondID || image.Prompt != "prompt 2" || image.Base64Data != "image 2" {
		t.Fatalf("expected second image after undo clear, got %#v", image)
	}

	image, err = database.RecordUndoToPreviousImage("sess_clear", SessionEvent{
		SessionID:       "sess_clear",
		EventType:       "undo",
		PreviousImageID: secondID,
	})
	if err != nil {
		t.Fatalf("failed to undo to first image: %v", err)
	}
	if image == nil || image.ImageID != firstID || image.Prompt != "prompt 1" || image.Base64Data != "image 1" {
		t.Fatalf("expected first image after second undo, got %#v", image)
	}
}

func TestClearIsVersionTreeNode(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(dir)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.UpsertSession("client_test", "sess_version_tree"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	firstID, err := database.RecordGeneratedImage("sess_version_tree", "prompt 1", "image 1", "title 1", "summary 1", SessionEvent{
		SessionID: "sess_version_tree",
		EventType: "image_generated",
		Dev:       "prompt 1",
	})
	if err != nil {
		t.Fatalf("failed to record first image: %v", err)
	}
	secondID, err := database.RecordGeneratedImage("sess_version_tree", "prompt 2", "image 2", "title 2", "summary 2", SessionEvent{
		SessionID: "sess_version_tree",
		EventType: "image_generated",
		Dev:       "prompt 2",
	})
	if err != nil {
		t.Fatalf("failed to record second image: %v", err)
	}

	secondVersionID := sessionVersionIDForImage(t, database, "sess_version_tree", secondID)
	if err := database.RecordClear("sess_version_tree", SessionEvent{
		SessionID:       "sess_version_tree",
		EventType:       "clear",
		PreviousImageID: secondID,
	}); err != nil {
		t.Fatalf("failed to record clear: %v", err)
	}

	var clearVersionID int64
	var clearParentID sql.NullInt64
	if err := database.conn.QueryRow(`
		SELECT id, parent_id
		FROM session_versions
		WHERE session_id = ? AND event_type = ?
		ORDER BY id DESC
		LIMIT 1
	`, "sess_version_tree", "clear").Scan(&clearVersionID, &clearParentID); err != nil {
		t.Fatalf("failed to query clear version: %v", err)
	}
	if !clearParentID.Valid || clearParentID.Int64 != secondVersionID {
		t.Fatalf("expected clear parent version %d, got %v", secondVersionID, clearParentID)
	}

	image, err := database.RecordUndoToPreviousImage("sess_version_tree", SessionEvent{
		SessionID:       "sess_version_tree",
		EventType:       "undo",
		PreviousImageID: 0,
	})
	if err != nil {
		t.Fatalf("failed to undo clear: %v", err)
	}
	if image == nil || image.ImageID != secondID {
		t.Fatalf("expected undo clear to restore second image, got %#v", image)
	}

	thirdID, err := database.RecordGeneratedImage("sess_version_tree", "prompt 3", "image 3", "title 3", "summary 3", SessionEvent{
		SessionID: "sess_version_tree",
		EventType: "image_generated",
		Dev:       "prompt 3",
	})
	if err != nil {
		t.Fatalf("failed to record third image: %v", err)
	}
	thirdVersionID := sessionVersionIDForImage(t, database, "sess_version_tree", thirdID)
	var thirdParentID sql.NullInt64
	if err := database.conn.QueryRow(`
		SELECT parent_id
		FROM session_versions
		WHERE id = ?
	`, thirdVersionID).Scan(&thirdParentID); err != nil {
		t.Fatalf("failed to query third version parent: %v", err)
	}
	if !thirdParentID.Valid || thirdParentID.Int64 != secondVersionID {
		t.Fatalf("expected third version to branch from restored second version %d, got %v", secondVersionID, thirdParentID)
	}

	if firstID == 0 {
		t.Fatal("expected first image id")
	}
}

func sessionVersionIDForImage(t *testing.T, database *DB, sessionID string, imageID int64) int64 {
	t.Helper()
	var versionID int64
	if err := database.conn.QueryRow(`
		SELECT id
		FROM session_versions
		WHERE session_id = ? AND image_id = ?
	`, sessionID, imageID).Scan(&versionID); err != nil {
		t.Fatalf("failed to query version for image %d: %v", imageID, err)
	}
	return versionID
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
