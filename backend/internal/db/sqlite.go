package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "voxcanvas.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			dev TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			current_image_id INTEGER,
			undo_image_id INTEGER,
			current_version_id INTEGER,
			undo_version_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS sentences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			previous_image_id INTEGER,
			content TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL,
			base64_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS session_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			sentence_id INTEGER,
			image_id INTEGER,
			previous_image_id INTEGER,
			sentence TEXT,
			dev TEXT,
			before_dev TEXT,
			before_image_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS session_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			parent_id INTEGER,
			event_type TEXT NOT NULL,
			image_id INTEGER,
			dev TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sentences", "session_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sessions", "dev", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sessions", "title", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sessions", "summary", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sessions", "current_image_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sessions", "undo_image_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sessions", "current_version_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sessions", "undo_version_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sentences", "previous_image_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "images", "session_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_events", "sentence_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_events", "image_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_events", "previous_image_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_events", "sentence", "TEXT"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_events", "dev", "TEXT"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_events", "before_dev", "TEXT"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_events", "before_image_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_versions", "parent_id", "INTEGER"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "session_versions", "image_id", "INTEGER"); err != nil {
		return nil, err
	}
	if _, err := conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_sessions_client_id ON sessions(client_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_client_updated ON sessions(client_id, updated_at);
		CREATE INDEX IF NOT EXISTS idx_sentences_session_id ON sentences(session_id);
		CREATE INDEX IF NOT EXISTS idx_images_session_id ON images(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_events_session_id ON session_events(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_events_type ON session_events(event_type);
		CREATE INDEX IF NOT EXISTS idx_session_versions_session_id ON session_versions(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_versions_parent_id ON session_versions(parent_id);
	`); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func ensureColumn(conn *sql.DB, table, column, definition string) error {
	rows, err := conn.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			typ        string
			notNull    int
			defaultVal interface{}
			pk         int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = conn.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
	return err
}

func (d *DB) UpsertSession(clientID, sessionID string) error {
	return d.withTx(func(tx *sql.Tx) error {
		return upsertSessionTx(tx, clientID, sessionID)
	})
}

func (d *DB) UpdateSessionDev(sessionID, dev string) error {
	return d.withTx(func(tx *sql.Tx) error {
		return updateSessionDevTx(tx, sessionID, dev)
	})
}

func (d *DB) UpdateSessionMeta(sessionID, title, summary string) error {
	return d.withTx(func(tx *sql.Tx) error {
		return updateSessionMetaTx(tx, sessionID, title, summary)
	})
}

func (d *DB) CurrentImageID(sessionID string) (int64, error) {
	var imageID sql.NullInt64
	err := d.conn.QueryRow("SELECT current_image_id FROM sessions WHERE id = ?", sessionID).Scan(&imageID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !imageID.Valid {
		return 0, nil
	}
	return imageID.Int64, nil
}

func (d *DB) RecordSentence(sessionID string, previousImageID int64, content, typ, beforeDev string) (int64, error) {
	var sentenceID int64
	err := d.withTx(func(tx *sql.Tx) error {
		var err error
		sentenceID, err = insertSentenceTx(tx, sessionID, previousImageID, content, typ)
		if err != nil {
			return err
		}
		return insertSessionEventTx(tx, SessionEvent{
			SessionID:       sessionID,
			EventType:       "sentence",
			SentenceID:      sentenceID,
			PreviousImageID: previousImageID,
			Sentence:        content,
			BeforeDev:       beforeDev,
			BeforeImageID:   previousImageID,
		})
	})
	return sentenceID, err
}

func (d *DB) RecordRequirementRefined(sessionID, title, summary string, event SessionEvent) error {
	return d.withTx(func(tx *sql.Tx) error {
		if err := updateSessionDevTx(tx, sessionID, event.Dev); err != nil {
			return err
		}
		if err := updateSessionMetaIfEmptyTx(tx, sessionID, title, summary); err != nil {
			return err
		}
		return insertSessionEventTx(tx, event)
	})
}

func (d *DB) RecordGeneratedImage(sessionID, prompt, base64Data, title, summary string, event SessionEvent) (int64, error) {
	var imageID int64
	err := d.withTx(func(tx *sql.Tx) error {
		var err error
		parentVersionID, err := currentVersionIDTx(tx, sessionID)
		if err != nil {
			return err
		}
		imageID, err = insertImageTx(tx, sessionID, prompt, base64Data)
		if err != nil {
			return err
		}
		versionID, err := insertSessionVersionTx(tx, sessionID, parentVersionID, "image_generated", imageID, prompt)
		if err != nil {
			return err
		}
		event.ImageID = imageID
		if err := insertSessionEventTx(tx, event); err != nil {
			return err
		}
		if err := updateSessionMetaIfEmptyTx(tx, sessionID, title, summary); err != nil {
			return err
		}
		if err := updateSessionCurrentImageTx(tx, sessionID, imageID); err != nil {
			return err
		}
		if err := updateSessionUndoImageTx(tx, sessionID, imageID); err != nil {
			return err
		}
		if err := updateSessionCurrentVersionTx(tx, sessionID, versionID); err != nil {
			return err
		}
		if err := updateSessionUndoVersionTx(tx, sessionID, versionID); err != nil {
			return err
		}
		return updateSessionDevTx(tx, sessionID, "")
	})
	return imageID, err
}

func (d *DB) RecordUndo(sessionID, dev, title, summary string, event SessionEvent) error {
	return d.withTx(func(tx *sql.Tx) error {
		if err := updateSessionDevTx(tx, sessionID, dev); err != nil {
			return err
		}
		return insertSessionEventTx(tx, event)
	})
}

type GeneratedImage struct {
	ImageID    int64
	Prompt     string
	Base64Data string
}

type SessionVersion struct {
	ID        int64
	ParentID  int64
	EventType string
	ImageID   int64
	Dev       string
}

func (d *DB) RecordUndoToPreviousImage(sessionID string, event SessionEvent) (*GeneratedImage, error) {
	var image *GeneratedImage
	err := d.withTx(func(tx *sql.Tx) error {
		targetVersionID, ok, err := undoVersionIDTx(tx, sessionID)
		if err != nil {
			return err
		}
		if ok {
			targetVersion, err := sessionVersionByIDTx(tx, sessionID, targetVersionID)
			if err != nil {
				return err
			}
			if targetVersion == nil {
				ok = false
			} else {
				image, err = restoreSessionVersionTx(tx, sessionID, targetVersion, event)
				return err
			}
		}

		var target *GeneratedImage
		targetImageID, ok, err := undoImageIDTx(tx, sessionID)
		if err != nil {
			return err
		}
		if ok {
			target, err = imageByIDTx(tx, sessionID, targetImageID)
		}
		if err != nil {
			return err
		}
		if target == nil {
			event.ImageID = 0
			event.Dev = ""
			if err := insertSessionEventTx(tx, event); err != nil {
				return err
			}
			image = nil
			return nil
		}

		event.ImageID = target.ImageID
		event.Dev = target.Prompt
		if err := updateSessionDevTx(tx, sessionID, target.Prompt); err != nil {
			return err
		}
		if err := updateSessionCurrentImageTx(tx, sessionID, target.ImageID); err != nil {
			return err
		}
		nextImageID, err := previousImageIDTx(tx, sessionID, target.ImageID)
		if err != nil {
			return err
		}
		if err := updateSessionUndoImageTx(tx, sessionID, nextImageID); err != nil {
			return err
		}
		if err := insertSessionEventTx(tx, event); err != nil {
			return err
		}

		image = target
		return nil
	})
	return image, err
}

func (d *DB) RecordClear(sessionID string, event SessionEvent) error {
	return d.withTx(func(tx *sql.Tx) error {
		parentVersionID, err := currentVersionIDTx(tx, sessionID)
		if err != nil {
			return err
		}
		versionID, err := insertSessionVersionTx(tx, sessionID, parentVersionID, "clear", 0, "")
		if err != nil {
			return err
		}
		if err := updateSessionDevTx(tx, sessionID, ""); err != nil {
			return err
		}
		if err := updateSessionCurrentImageTx(tx, sessionID, 0); err != nil {
			return err
		}
		if err := updateSessionUndoImageTx(tx, sessionID, event.PreviousImageID); err != nil {
			return err
		}
		if err := updateSessionCurrentVersionTx(tx, sessionID, versionID); err != nil {
			return err
		}
		if err := updateSessionUndoVersionTx(tx, sessionID, parentVersionID); err != nil {
			return err
		}
		return insertSessionEventTx(tx, event)
	})
}

func (d *DB) RecordSwitchSession(clientID, newSessionID string, event SessionEvent) error {
	return d.withTx(func(tx *sql.Tx) error {
		if err := upsertSessionTx(tx, clientID, newSessionID); err != nil {
			return err
		}
		return insertSessionEventTx(tx, event)
	})
}

type SessionEvent struct {
	SessionID       string
	EventType       string
	SentenceID      int64
	ImageID         int64
	PreviousImageID int64
	Sentence        string
	Dev             string
	BeforeDev       string
	BeforeImageID   int64
}

type SessionSummary struct {
	SessionID string
	Title     string
	Summary   string
	Dev       string
	UpdatedAt string
}

func (d *DB) ListSessionsByClient(clientID string, limit int) ([]SessionSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := d.conn.Query(`
		SELECT id, title, summary, dev, updated_at
		FROM sessions
		WHERE client_id = ?
		ORDER BY updated_at DESC
		LIMIT ?
	`, clientID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]SessionSummary, 0)
	for rows.Next() {
		var session SessionSummary
		if err := rows.Scan(&session.SessionID, &session.Title, &session.Summary, &session.Dev, &session.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (d *DB) InsertSessionEvent(event SessionEvent) error {
	return d.withTx(func(tx *sql.Tx) error {
		return insertSessionEventTx(tx, event)
	})
}

func (d *DB) withTx(fn func(*sql.Tx) error) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertSessionTx(tx *sql.Tx, clientID, sessionID string) error {
	_, err := tx.Exec(`
		INSERT INTO sessions (id, client_id, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			client_id = excluded.client_id,
			updated_at = CURRENT_TIMESTAMP
	`, sessionID, clientID)
	return err
}

func updateSessionDevTx(tx *sql.Tx, sessionID, dev string) error {
	_, err := tx.Exec(`
		UPDATE sessions
		SET dev = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, dev, sessionID)
	return err
}

func updateSessionMetaTx(tx *sql.Tx, sessionID, title, summary string) error {
	_, err := tx.Exec(`
		UPDATE sessions
		SET title = ?, summary = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, title, summary, sessionID)
	return err
}

func updateSessionMetaIfEmptyTx(tx *sql.Tx, sessionID, title, summary string) error {
	_, err := tx.Exec(`
		UPDATE sessions
		SET
			title = CASE WHEN title = '' THEN ? ELSE title END,
			summary = CASE WHEN summary = '' THEN ? ELSE summary END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, title, summary, sessionID)
	return err
}

func updateSessionCurrentImageTx(tx *sql.Tx, sessionID string, imageID int64) error {
	_, err := tx.Exec(`
		UPDATE sessions
		SET current_image_id = NULLIF(?, 0), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, imageID, sessionID)
	return err
}

func updateSessionUndoImageTx(tx *sql.Tx, sessionID string, imageID int64) error {
	_, err := tx.Exec(`
		UPDATE sessions
		SET undo_image_id = NULLIF(?, 0), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, imageID, sessionID)
	return err
}

func updateSessionCurrentVersionTx(tx *sql.Tx, sessionID string, versionID int64) error {
	_, err := tx.Exec(`
		UPDATE sessions
		SET current_version_id = NULLIF(?, 0), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, versionID, sessionID)
	return err
}

func updateSessionUndoVersionTx(tx *sql.Tx, sessionID string, versionID int64) error {
	_, err := tx.Exec(`
		UPDATE sessions
		SET undo_version_id = NULLIF(?, 0), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, versionID, sessionID)
	return err
}

func currentImageIDTx(tx *sql.Tx, sessionID string) (int64, error) {
	imageID, ok, err := nullableImageIDTx(tx, "SELECT current_image_id FROM sessions WHERE id = ?", sessionID)
	if err != nil || !ok {
		return 0, err
	}
	return imageID, nil
}

func undoImageIDTx(tx *sql.Tx, sessionID string) (int64, bool, error) {
	imageID, ok, err := nullableImageIDTx(tx, "SELECT undo_image_id FROM sessions WHERE id = ?", sessionID)
	return imageID, ok, err
}

func currentVersionIDTx(tx *sql.Tx, sessionID string) (int64, error) {
	versionID, ok, err := nullableImageIDTx(tx, "SELECT current_version_id FROM sessions WHERE id = ?", sessionID)
	if err != nil || !ok {
		return 0, err
	}
	return versionID, nil
}

func undoVersionIDTx(tx *sql.Tx, sessionID string) (int64, bool, error) {
	versionID, ok, err := nullableImageIDTx(tx, "SELECT undo_version_id FROM sessions WHERE id = ?", sessionID)
	return versionID, ok, err
}

func nullableImageIDTx(tx *sql.Tx, query, sessionID string) (int64, bool, error) {
	var imageID sql.NullInt64
	err := tx.QueryRow(query, sessionID).Scan(&imageID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !imageID.Valid {
		return 0, false, nil
	}
	return imageID.Int64, true, nil
}

func insertSessionVersionTx(tx *sql.Tx, sessionID string, parentID int64, eventType string, imageID int64, dev string) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO session_versions (session_id, parent_id, event_type, image_id, dev)
		VALUES (?, NULLIF(?, 0), ?, NULLIF(?, 0), ?)
	`, sessionID, parentID, eventType, imageID, dev)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func sessionVersionByIDTx(tx *sql.Tx, sessionID string, versionID int64) (*SessionVersion, error) {
	var (
		version  SessionVersion
		parentID sql.NullInt64
		imageID  sql.NullInt64
	)
	err := tx.QueryRow(`
		SELECT id, parent_id, event_type, image_id, dev
		FROM session_versions
		WHERE session_id = ? AND id = ?
	`, sessionID, versionID).Scan(&version.ID, &parentID, &version.EventType, &imageID, &version.Dev)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		version.ParentID = parentID.Int64
	}
	if imageID.Valid {
		version.ImageID = imageID.Int64
	}
	return &version, nil
}

func restoreSessionVersionTx(tx *sql.Tx, sessionID string, version *SessionVersion, event SessionEvent) (*GeneratedImage, error) {
	restore := &GeneratedImage{
		ImageID: version.ImageID,
		Prompt:  version.Dev,
	}
	if version.ImageID != 0 {
		image, err := imageByIDTx(tx, sessionID, version.ImageID)
		if err != nil || image == nil {
			return nil, err
		}
		restore = image
	}

	event.ImageID = restore.ImageID
	event.Dev = restore.Prompt
	if err := updateSessionDevTx(tx, sessionID, restore.Prompt); err != nil {
		return nil, err
	}
	if err := updateSessionCurrentImageTx(tx, sessionID, restore.ImageID); err != nil {
		return nil, err
	}
	if err := updateSessionCurrentVersionTx(tx, sessionID, version.ID); err != nil {
		return nil, err
	}
	if err := updateSessionUndoVersionTx(tx, sessionID, version.ParentID); err != nil {
		return nil, err
	}
	nextImageID, err := nearestImageIDFromVersionTx(tx, sessionID, version.ParentID)
	if err != nil {
		return nil, err
	}
	if err := updateSessionUndoImageTx(tx, sessionID, nextImageID); err != nil {
		return nil, err
	}
	if err := insertSessionEventTx(tx, event); err != nil {
		return nil, err
	}
	return restore, nil
}

func nearestImageIDFromVersionTx(tx *sql.Tx, sessionID string, versionID int64) (int64, error) {
	for versionID != 0 {
		version, err := sessionVersionByIDTx(tx, sessionID, versionID)
		if err != nil || version == nil {
			return 0, err
		}
		if version.ImageID != 0 {
			return version.ImageID, nil
		}
		versionID = version.ParentID
	}
	return 0, nil
}

func imageByIDTx(tx *sql.Tx, sessionID string, imageID int64) (*GeneratedImage, error) {
	return queryGeneratedImageTx(tx, `
		SELECT id, prompt, base64_data
		FROM images
		WHERE session_id = ? AND id = ?
		LIMIT 1
	`, sessionID, imageID)
}

func latestImageTx(tx *sql.Tx, sessionID string) (*GeneratedImage, error) {
	return queryGeneratedImageTx(tx, `
		SELECT id, prompt, base64_data
		FROM images
		WHERE session_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, sessionID)
}

func previousImageTx(tx *sql.Tx, sessionID string, currentImageID int64) (*GeneratedImage, error) {
	return queryGeneratedImageTx(tx, `
		SELECT id, prompt, base64_data
		FROM images
		WHERE session_id = ? AND id < ?
		ORDER BY id DESC
		LIMIT 1
	`, sessionID, currentImageID)
}

func previousImageIDTx(tx *sql.Tx, sessionID string, currentImageID int64) (int64, error) {
	image, err := previousImageTx(tx, sessionID, currentImageID)
	if err != nil || image == nil {
		return 0, err
	}
	return image.ImageID, nil
}

func queryGeneratedImageTx(tx *sql.Tx, query string, args ...interface{}) (*GeneratedImage, error) {
	var image GeneratedImage
	err := tx.QueryRow(query, args...).Scan(&image.ImageID, &image.Prompt, &image.Base64Data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func sessionMetaFromText(text string) (string, string) {
	summary := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if summary == "" {
		return "", ""
	}
	return truncateRunes(summary, 18), truncateRunes(summary, 80)
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

func insertSentenceTx(tx *sql.Tx, sessionID string, previousImageID int64, content, typ string) (int64, error) {
	result, err := tx.Exec(
		"INSERT INTO sentences (session_id, previous_image_id, content, type) VALUES (?, NULLIF(?, 0), ?, ?)",
		sessionID,
		previousImageID,
		content,
		typ,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func insertImageTx(tx *sql.Tx, sessionID, prompt, base64Data string) (int64, error) {
	result, err := tx.Exec("INSERT INTO images (session_id, prompt, base64_data) VALUES (?, ?, ?)", sessionID, prompt, base64Data)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func insertSessionEventTx(tx *sql.Tx, event SessionEvent) error {
	log.Printf("[DB] session_event session_id=%s event_type=%s image_id=%d previous_image_id=%d", event.SessionID, event.EventType, event.ImageID, event.PreviousImageID)
	_, err := tx.Exec(`
		INSERT INTO session_events (
			session_id,
			event_type,
			sentence_id,
			image_id,
			previous_image_id,
			sentence,
			dev,
			before_dev,
			before_image_id
		)
		VALUES (?, ?, NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0), ?, ?, ?, NULLIF(?, 0))
	`,
		event.SessionID,
		event.EventType,
		event.SentenceID,
		event.ImageID,
		event.PreviousImageID,
		event.Sentence,
		event.Dev,
		event.BeforeDev,
		event.BeforeImageID,
	)
	return err
}

func (d *DB) Close() error {
	return d.conn.Close()
}
