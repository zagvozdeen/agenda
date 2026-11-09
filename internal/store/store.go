package store

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

type Storage interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context)

	CreateUser(ctx context.Context, username, name string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	FindUsers(ctx context.Context, query string, limit, offset int) ([]models.User, error)
	DisableUser(ctx context.Context, username string) error

	CreateToken(ctx context.Context, token *models.APIToken) error
	ListTokens(ctx context.Context, userID uuid.UUID) ([]models.APIToken, error)
	RevokeToken(ctx context.Context, id uuid.UUID) error
	AuthenticateToken(ctx context.Context, hash string) (*models.User, error)

	GetBoard(ctx context.Context, key string, lock bool) (*models.Board, error)
	ListBoards(ctx context.Context, memberID *uuid.UUID, limit, offset int) ([]models.Board, error)
	CreateBoard(ctx context.Context, board *models.Board) error
	UpdateBoard(ctx context.Context, board *models.Board) error
	DeleteBoard(ctx context.Context, key string) error
	CountTasksByStatus(ctx context.Context, key string) (map[enums.TaskStatus]int, error)

	ListMemberships(ctx context.Context, userID uuid.UUID) ([]models.Membership, error)
	GetBoardRole(ctx context.Context, key string, userID uuid.UUID) (enums.BoardRole, error)
	CountOwners(ctx context.Context, key string) (int, error)
	ListMembers(ctx context.Context, key string) ([]models.Member, error)
	SaveMember(ctx context.Context, member *models.BoardMember) error
	RemoveMember(ctx context.Context, key string, userID uuid.UUID) error

	ListSprints(ctx context.Context, key string, status *enums.SprintStatus, limit, offset int) ([]models.Sprint, error)
	GetSprint(ctx context.Context, key string, number int) (*models.Sprint, error)
	GetActiveSprint(ctx context.Context, key string) (*models.Sprint, error)
	CreateSprint(ctx context.Context, sprint *models.Sprint) error
	UpdateSprint(ctx context.Context, sprint *models.Sprint) error

	GetTask(ctx context.Context, id string) (*models.Task, error)
	TaskExists(ctx context.Context, id string) (bool, error)
	LockIdempotency(ctx context.Context, key uuid.UUID) error
	GetTaskByIdempotency(ctx context.Context, key uuid.UUID) (*models.Task, error)
	CreateTask(ctx context.Context, task *models.Task) error
	UpdateTask(ctx context.Context, task *models.Task) error
	DeleteTask(ctx context.Context, task *models.Task) error
	HasTaskChildren(ctx context.Context, id string) (bool, error)
	ParentCreatesCycle(ctx context.Context, parentID, taskID string) (bool, error)
	ListSubtasks(ctx context.Context, id string) ([]models.Task, error)
	ListUnfinishedTasks(ctx context.Context, key string, number int) ([]models.Task, error)
	SearchTasks(ctx context.Context, filter TaskFilter) ([]models.Task, error)

	GetTaskLink(ctx context.Context, from, to string, kind enums.LinkType) (*models.TaskLink, error)
	ListTaskLinks(ctx context.Context, id string) ([]models.TaskLink, error)
	CreateTaskLink(ctx context.Context, link *models.TaskLink) error
	DeleteTaskLink(ctx context.Context, from, to string, kind enums.LinkType) error

	GetComment(ctx context.Context, id uuid.UUID) (*models.Comment, error)
	ListComments(ctx context.Context, taskID string) ([]models.Comment, error)
	CreateComment(ctx context.Context, comment *models.Comment) error
	UpdateComment(ctx context.Context, comment *models.Comment) error
	DeleteComment(ctx context.Context, comment *models.Comment) error

	CreateTaskEvent(ctx context.Context, event *models.TaskEvent) error
	ListActivity(ctx context.Context, filter ActivityFilter) ([]models.TaskEvent, error)
}

type Store struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

var _ Storage = (*Store)(nil)

func New(pool *pgxpool.Pool, log *slog.Logger) *Store {
	return &Store{pool: pool, log: log}
}
