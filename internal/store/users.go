package store

import (
	"context"
	"errors"
	"fmt"
	"uuid"

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
		"INSERT INTO users (username, name, created_at) VALUES ($1, $2, now()) RETURNING id, username, name, created_at, deleted_at",
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
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var user models.User
		err = rows.Scan(&user.ID, &user.Username, &user.Name, &user.CreatedAt, &user.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}
		users = append(users, user)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
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

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT id, username, name, created_at, deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL",
		id,
	).Scan(&user.ID, &user.Username, &user.Name, &user.CreatedAt, &user.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return user, nil
}

func (s *Store) FindUsers(ctx context.Context, query string, limit, offset int) ([]models.User, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT id, username, name, created_at, deleted_at FROM users WHERE deleted_at IS NULL AND (strpos(lower(name), lower($1)) > 0 OR strpos(lower(username), lower($1)) > 0) ORDER BY username LIMIT $2 OFFSET $3",
		query, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var user models.User
		err = rows.Scan(&user.ID, &user.Username, &user.Name, &user.CreatedAt, &user.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to find users: %w", err)
		}
		users = append(users, user)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	return users, nil
}
