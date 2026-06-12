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
	query  string
	args   []interface{}
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
		CREATE TABLE IF NOT EXISTS sentences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			prompt TEXT NOT NULL,
			base64_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return nil, err
	}

	d := &DB{
		conn:  conn,
		queue: make(chan writeJob, 256),
	}
	go d.worker()
	return d, nil
}

func (d *DB) worker() {
	for job := range d.queue {
		if _, err := d.conn.Exec(job.query, job.args...); err != nil {
			log.Printf("db write error: %v", err)
		}
	}
}

func (d *DB) InsertSentence(content, typ string) {
	d.queue <- writeJob{
		query: "INSERT INTO sentences (content, type) VALUES (?, ?)",
		args:  []interface{}{content, typ},
	}
}

func (d *DB) InsertImage(prompt, base64Data string) {
	d.queue <- writeJob{
		query: "INSERT INTO images (prompt, base64_data) VALUES (?, ?)",
		args:  []interface{}{prompt, base64Data},
	}
}

func (d *DB) Close() error {
	close(d.queue)
	return d.conn.Close()
}
