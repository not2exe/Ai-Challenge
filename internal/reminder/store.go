package reminder

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store provides SQLite-backed storage for reminders.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database at dbPath and
// ensures the reminders table exists.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if err := createTable(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reminders (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			title       TEXT    NOT NULL,
			description TEXT    NOT NULL DEFAULT '',
			due_date    TEXT    NOT NULL,
			priority    TEXT    NOT NULL DEFAULT 'medium',
			status      TEXT    NOT NULL DEFAULT 'pending',
			created_at  TEXT    NOT NULL,
			updated_at  TEXT    NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Add inserts a new reminder and returns it with the assigned ID.
func (s *Store) Add(r Reminder) (*Reminder, error) {
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	if r.Priority == "" {
		r.Priority = PriorityMedium
	}
	if r.Status == "" {
		r.Status = StatusPending
	}

	result, err := s.db.Exec(`
		INSERT INTO reminders (title, description, due_date, priority, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, r.Title, r.Description, r.DueDate.UTC().Format(time.RFC3339),
		r.Priority, r.Status,
		r.CreatedAt.Format(time.RFC3339), r.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to insert reminder: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted ID: %w", err)
	}
	r.ID = id

	return &r, nil
}

// List returns all reminders, optionally filtered by status.
// Pass an empty string to list all.
func (s *Store) List(statusFilter string) ([]Reminder, error) {
	var rows *sql.Rows
	var err error

	if statusFilter != "" {
		rows, err = s.db.Query(`
			SELECT id, title, description, due_date, priority, status, created_at, updated_at
			FROM reminders WHERE status = ? ORDER BY due_date ASC
		`, statusFilter)
	} else {
		rows, err = s.db.Query(`
			SELECT id, title, description, due_date, priority, status, created_at, updated_at
			FROM reminders ORDER BY due_date ASC
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	defer rows.Close()

	return scanReminders(rows)
}

// GetDue returns all pending reminders whose due_date is at or before now.
func (s *Store) GetDue() ([]Reminder, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	rows, err := s.db.Query(`
		SELECT id, title, description, due_date, priority, status, created_at, updated_at
		FROM reminders WHERE status = ? AND due_date <= ? ORDER BY due_date ASC
	`, StatusPending, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get due reminders: %w", err)
	}
	defer rows.Close()

	return scanReminders(rows)
}

// GetByID returns a single reminder by ID.
func (s *Store) GetByID(id int64) (*Reminder, error) {
	row := s.db.QueryRow(`
		SELECT id, title, description, due_date, priority, status, created_at, updated_at
		FROM reminders WHERE id = ?
	`, id)

	r, err := scanReminder(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reminder %d not found", id)
		}
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}
	return r, nil
}

// Complete marks a reminder as completed.
func (s *Store) Complete(id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)

	result, err := s.db.Exec(`
		UPDATE reminders SET status = ?, updated_at = ? WHERE id = ?
	`, StatusCompleted, now, id)
	if err != nil {
		return fmt.Errorf("failed to complete reminder: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("reminder %d not found", id)
	}
	return nil
}

// Delete removes a reminder by ID.
func (s *Store) Delete(id int64) error {
	result, err := s.db.Exec(`DELETE FROM reminders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("reminder %d not found", id)
	}
	return nil
}

// UpdateFields holds optional fields for a partial update.
type UpdateFields struct {
	Title       *string
	Description *string
	DueDate     *time.Time
	Priority    *string
}

// Update applies partial updates to a reminder.
func (s *Store) Update(id int64, fields UpdateFields) (*Reminder, error) {
	// Build SET clause dynamically
	setClauses := []string{}
	args := []interface{}{}

	if fields.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *fields.Title)
	}
	if fields.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *fields.Description)
	}
	if fields.DueDate != nil {
		setClauses = append(setClauses, "due_date = ?")
		args = append(args, fields.DueDate.UTC().Format(time.RFC3339))
	}
	if fields.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *fields.Priority)
	}

	if len(setClauses) == 0 {
		return s.GetByID(id)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, now)

	query := "UPDATE reminders SET "
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += " WHERE id = ?"
	args = append(args, id)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update reminder: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("reminder %d not found", id)
	}

	return s.GetByID(id)
}

// scanReminders reads multiple rows into a slice of Reminder.
func scanReminders(rows *sql.Rows) ([]Reminder, error) {
	var reminders []Reminder
	for rows.Next() {
		var r Reminder
		var dueDate, createdAt, updatedAt string

		if err := rows.Scan(&r.ID, &r.Title, &r.Description,
			&dueDate, &r.Priority, &r.Status,
			&createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}

		r.DueDate, _ = time.Parse(time.RFC3339, dueDate)
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		reminders = append(reminders, r)
	}
	return reminders, rows.Err()
}

// scanReminder reads a single row into a Reminder.
func scanReminder(row *sql.Row) (*Reminder, error) {
	var r Reminder
	var dueDate, createdAt, updatedAt string

	if err := row.Scan(&r.ID, &r.Title, &r.Description,
		&dueDate, &r.Priority, &r.Status,
		&createdAt, &updatedAt); err != nil {
		return nil, err
	}

	r.DueDate, _ = time.Parse(time.RFC3339, dueDate)
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &r, nil
}
