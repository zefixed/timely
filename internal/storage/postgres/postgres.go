package postgres

import (
	"database/sql"
	"fmt"
	"time"
	"timely/internal/storage"

	"github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

type DBAction struct {
	UserID          int       `json:"user_id"`
	UserName        string    `json:"user_name"`
	UserDescription string    `json:"user_description"`
	EventDatetime   time.Time `json:"event_datetime"`
	Action          string    `json:"action"`
}

type User struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// TODO: Switch to migrations
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255)
		);
		CREATE TABLE IF NOT EXISTS attendance (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL,
			event_datetime TIMESTAMP NOT NULL,
			action VARCHAR(20) CHECK (action IN ('in', 'out')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE);
        CREATE INDEX IF NOT EXISTS idx_attendance_event_datetime ON attendance(event_datetime);
        CREATE INDEX IF NOT EXISTS idx_attendance_user_event_datetime ON attendance(user_id, event_datetime);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) CreateUser(name, desc string) error {
	const op = "storage.postgres.CreateUser"

	stmt, err := s.db.Prepare("INSERT INTO users(name, description) VALUES($1, $2)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(name, desc)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetUser(id int) ([]User, error) {
	const op = "storage.postgres.GetUser"

	query := `SELECT * FROM users`
	if id != -1 {
		query += ` WHERE id = $1`
	}

	stmt, err := s.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var rows *sql.Rows
	if id != -1 {
		rows, err = stmt.Query(id)
	} else {
		rows, err = stmt.Query()
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var users []User

	if !rows.Next() {
		return nil, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}

	for {
		var user User
		err = rows.Scan(
			&user.ID,
			&user.Name,
			&user.Description,
		)

		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		users = append(users, user)

		if !rows.Next() {
			break
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return users, nil
}

func (s *Storage) DeleteUser(id int) error {
	const op = "storage.postgres.DeleteUser"

	stmt, err := s.db.Prepare("DELETE FROM users WHERE id = $1")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// checking whether the user was removed
	rowAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}

	return nil
}

func (s *Storage) CreateAction(id int, action string) error {
	const op = "storage.postgres.CreateAction"

	if action != "in" && action != "out" {
		return fmt.Errorf("%s: %w", op, storage.ErrIncorrectAction)
	}

	stmt, err := s.db.Prepare("INSERT INTO attendance(user_id, event_datetime, action) VALUES($1, $2, $3)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(id, time.Now(), action)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" {
				return fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
			} else {
				return fmt.Errorf("%s: %w", op, err)
			}
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetActions(id int, eventsStart, eventsEnd time.Time, action string) ([]DBAction, error) {
	// set id = -1 to ignore user
	// set actions = "" to ignore action

	const op = "storage.postgres.GetAction"

	if action != "in" && action != "out" && action != "" {
		return nil, fmt.Errorf("%s: %w", op, storage.ErrIncorrectAction)
	}

	query := `
			SELECT 
				u.id AS user_id, u.name AS user_name, u.description AS user_description,
				a.event_datetime AS event_datetime, a.action AS action
			FROM users u
			JOIN attendance a ON u.id = a.user_id
			WHERE event_datetime BETWEEN $1 AND $2
		`

	// TODO: get rid of duplication if statement
	if id != -1 && action != "" {
		query += ` AND u.id = $3 AND a.action = $4`
	} else if id != -1 && action == "" {
		query += ` AND u.id = $3`
	} else if id == -1 && action != "" {
		query += ` AND a.action = $3`
	}

	stmt, err := s.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var rows *sql.Rows
	if id != -1 && action != "" {
		rows, err = stmt.Query(eventsStart, eventsEnd, id, action)
	} else if id != -1 && action == "" {
		rows, err = stmt.Query(eventsStart, eventsEnd, id)
	} else if id == -1 && action != "" {
		rows, err = stmt.Query(eventsStart, eventsEnd, action)
	} else {
		rows, err = stmt.Query(eventsStart, eventsEnd)
	}

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" {
				return nil, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
			} else {
				return nil, fmt.Errorf("%s: %w", op, err)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var actions []DBAction
	for rows.Next() {
		var action DBAction
		err = rows.Scan(
			&action.UserID,
			&action.UserName,
			&action.UserDescription,
			&action.EventDatetime,
			&action.Action,
		)

		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		actions = append(actions, action)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return actions, nil
}
