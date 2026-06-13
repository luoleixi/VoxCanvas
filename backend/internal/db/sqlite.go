package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

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
	if _, err := conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_sessions_client_id ON sessions(client_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_client_updated ON sessions(client_id, updated_at);
		CREATE INDEX IF NOT EXISTS idx_sentences_session_id ON sentences(session_id);
		CREATE INDEX IF NOT EXISTS idx_images_session_id ON images(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_events_session_id ON session_events(session_id);
		CREATE INDEX IF NOT EXISTS idx_session_events_type ON session_events(event_type);
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
		if err := updateSessionMetaTx(tx, sessionID, title, summary); err != nil {
			return err
		}
		return insertSessionEventTx(tx, event)
	})
}

func (d *DB) RecordGeneratedImage(sessionID, prompt, base64Data, title, summary string, event SessionEvent) (int64, error) {
	var imageID int64
	err := d.withTx(func(tx *sql.Tx) error {
		var err error
		imageID, err = insertImageTx(tx, sessionID, prompt, base64Data)
		if err != nil {
			return err
		}
		event.ImageID = imageID
		if err := insertSessionEventTx(tx, event); err != nil {
			return err
		}
		if err := updateSessionMetaTx(tx, sessionID, title, summary); err != nil {
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
		if err := updateSessionMetaTx(tx, sessionID, title, summary); err != nil {
			return err
		}
		return insertSessionEventTx(tx, event)
	})
}

func (d *DB) RecordClear(sessionID string, event SessionEvent) error {
	return d.withTx(func(tx *sql.Tx) error {
		if err := updateSessionDevTx(tx, sessionID, ""); err != nil {
			return err
		}
		if err := updateSessionMetaTx(tx, sessionID, "", ""); err != nil {
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
