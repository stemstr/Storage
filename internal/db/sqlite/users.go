package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Pubkey string
}

type User struct {
	ID        string
	Pubkey    string
	CreatedAt time.Time
}

func (r *repo) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO users (id, pubkey, created_at) VALUES (?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	user := &User{
		ID:        uuid.New().String(),
		Pubkey:    req.Pubkey,
		CreatedAt: time.Now(),
	}

	if _, err := stmt.ExecContext(ctx, user.ID, user.Pubkey, user.CreatedAt); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *repo) GetUser(ctx context.Context, pubkey string) (*User, error) {
	var user User

	row := r.db.QueryRowContext(ctx, "SELECT id, pubkey, created_at FROM users WHERE pubkey=?", pubkey)

	err := row.Scan(&user.ID, &user.Pubkey, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *repo) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, pubkey, created_at FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Pubkey, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
