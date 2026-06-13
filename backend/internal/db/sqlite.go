package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn  *sql.DB
	queue chan writeJob
}

type writeJob struct {
	query string
	args  []interface{}
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS sentences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
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
		CREATE INDEX IF NOT EXISTS idx_sessions_client_id ON sessions(client_id);
		CREATE INDEX IF NOT EXISTS idx_sentences_session_id ON sentences(session_id);
		CREATE INDEX IF NOT EXISTS idx_images_session_id ON images(session_id);
	`); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "sentences", "session_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, err
	}
	if err := ensureColumn(conn, "images", "session_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, err
	}

	d := &DB{
		conn:  conn,
		queue: make(chan writeJob, 256),
	}
	go d.worker()
	return d, nil
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

func (d *DB) worker() {
	for job := range d.queue {
		if _, err := d.conn.Exec(job.query, job.args...); err != nil {
			log.Printf("db write error: %v", err)
		}
	}
}

func (d *DB) UpsertSession(clientID, sessionID string) error {
	_, err := d.conn.Exec(`
		INSERT INTO sessions (id, client_id, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			client_id = excluded.client_id,
			updated_at = CURRENT_TIMESTAMP
	`, sessionID, clientID)
	return err
}

func (d *DB) InsertSentence(sessionID, content, typ string) {
	d.queue <- writeJob{
		query: "INSERT INTO sentences (session_id, content, type) VALUES (?, ?, ?)",
		args:  []interface{}{sessionID, content, typ},
	}
}

func (d *DB) InsertImage(sessionID, prompt, base64Data string) {
	d.queue <- writeJob{
		query: "INSERT INTO images (session_id, prompt, base64_data) VALUES (?, ?, ?)",
		args:  []interface{}{sessionID, prompt, base64Data},
	}
}

func (d *DB) Close() error {
	close(d.queue)
	return d.conn.Close()
}
