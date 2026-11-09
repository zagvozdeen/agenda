package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
)

func (s *Store) CreateUser(ctx context.Context, username, name string) (*models.User, error) {
	u := &models.User{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO users (username, name) VALUES ($1, $2) RETURNING id, username, name, created_at, deleted_at",
		username, name,
	).Scan(&u.ID, &u.Username, &u.Name, &u.CreatedAt, &u.DeletedAt)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.ConstraintName == "users_username_key" {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return u, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	u := &models.User{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT id, username, name, created_at, deleted_at FROM users WHERE username = $1 AND deleted_at IS NULL",
		username,
	).Scan(&u.ID, &u.Username, &u.Name, &u.CreatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT id, username, name, created_at, deleted_at FROM users ORDER BY username",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.User])
}

func (s *Store) DisableUser(ctx context.Context, username string) error {
	result, err := s.querier(ctx).Exec(
		ctx,
		"UPDATE users SET deleted_at = now() WHERE username = $1 AND deleted_at IS NULL",
		username,
	)
	if err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
