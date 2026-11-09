package store

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
)

type Storage interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context)

	CreateUser(ctx context.Context, username string, name string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	DisableUser(ctx context.Context, username string) error

	CreateToken(ctx context.Context, token *models.APIToken) error
	ListTokens(ctx context.Context, uuid uuid.UUID) ([]models.APIToken, error)
	RevokeToken(ctx context.Context, uuid uuid.UUID) error
	AuthenticateToken(ctx context.Context, token string) (*models.User, error)
}

type Store struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

var _ Storage = (*Store)(nil)

func New(pool *pgxpool.Pool, log *slog.Logger) *Store {
	return &Store{pool: pool, log: log}
}
