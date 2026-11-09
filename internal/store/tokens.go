package store

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
)

var ErrInvalidToken = errors.New("invalid token")

func (s *Store) CreateToken(ctx context.Context, token *models.APIToken) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO api_tokens (user_id, token_hash, name, expires_at) SELECT id, $2, $3, $4 FROM users WHERE id = $1 AND deleted_at IS NULL RETURNING id, created_at",
		token.UserID, token.TokenHash, token.Name, token.ExpiresAt,
	).Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to create token: %w", err)
	}
	return nil
}

func (s *Store) ListTokens(ctx context.Context, userID uuid.UUID) ([]models.APIToken, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT id, user_id, name, created_at, last_used_at, expires_at, revoked_at FROM api_tokens WHERE user_id = $1 ORDER BY created_at, id",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByNameLax[models.APIToken])
}

func (s *Store) RevokeToken(ctx context.Context, id uuid.UUID) error {
	result, err := s.querier(ctx).Exec(
		ctx,
		"UPDATE api_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInvalidToken
	}
	return nil
}

func (s *Store) AuthenticateToken(ctx context.Context, hash string) (*models.User, error) {
	u := &models.User{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE api_tokens AS t SET last_used_at = now() FROM users AS u WHERE t.token_hash = $1 AND t.user_id = u.id AND t.revoked_at IS NULL AND (t.expires_at IS NULL OR t.expires_at > now()) AND u.deleted_at IS NULL RETURNING u.id, u.username, u.name, u.created_at, u.deleted_at",
		hash,
	).Scan(&u.ID, &u.Username, &u.Name, &u.CreatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to authenticate token: %w", err)
	}
	return u, nil
}
