package service

import "testing"

func TestGeneratedStoreClearCanBeRestoredByUndo(t *testing.T) {
	store := NewGeneratedStore()
	store.Set("sess_test", GeneratedResult{ImageID: 1, Text: "prompt 1", Image: "image 1"})
	store.Set("sess_test", GeneratedResult{ImageID: 2, Text: "prompt 2", Image: "image 2"})

	store.Clear("sess_test")

	if _, ok := store.Get("sess_test"); ok {
		t.Fatal("expected no current generated result after clear")
	}

	result, ok := store.UndoPrevious("sess_test")
	if !ok {
		t.Fatal("expected undo to restore cleared result")
	}
	if result.ImageID != 2 || result.Text != "prompt 2" || result.Image != "image 2" {
		t.Fatalf("expected second result after undo clear, got %#v", result)
	}

	result, ok = store.UndoPrevious("sess_test")
	if !ok {
		t.Fatal("expected second undo to restore older result")
	}
	if result.ImageID != 1 || result.Text != "prompt 1" || result.Image != "image 1" {
		t.Fatalf("expected first result after second undo, got %#v", result)
	}
}
