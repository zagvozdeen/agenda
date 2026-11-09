package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

var ErrSprintNotFound = errors.New("sprint not found")

const sprintColumns = "board_key, number, name, goal, status, start_date, end_date, created_at"

func (s *Store) ListSprints(ctx context.Context, key string, status *enums.SprintStatus, limit, offset int) ([]models.Sprint, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT "+sprintColumns+" FROM sprints WHERE board_key = $1 AND ($2::sprint_status IS NULL OR status = $2) ORDER BY number DESC LIMIT $3 OFFSET $4",
		key, status, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}
	defer rows.Close()
	var sprints []models.Sprint
	for rows.Next() {
		var sprint models.Sprint
		err = rows.Scan(&sprint.BoardKey, &sprint.Number, &sprint.Name, &sprint.Goal, &sprint.Status, &sprint.StartDate, &sprint.EndDate, &sprint.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list sprints: %w", err)
		}
		sprints = append(sprints, sprint)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}
	return sprints, nil
}

func (s *Store) GetSprint(ctx context.Context, key string, number int) (*models.Sprint, error) {
	sprint := &models.Sprint{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT "+sprintColumns+" FROM sprints WHERE board_key = $1 AND number = $2",
		key, number,
	).Scan(&sprint.BoardKey, &sprint.Number, &sprint.Name, &sprint.Goal, &sprint.Status, &sprint.StartDate, &sprint.EndDate, &sprint.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSprintNotFound
		}
		return nil, fmt.Errorf("failed to get sprint: %w", err)
	}
	return sprint, nil
}

func (s *Store) GetActiveSprint(ctx context.Context, key string) (*models.Sprint, error) {
	sprint := &models.Sprint{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT "+sprintColumns+" FROM sprints WHERE board_key = $1 AND status = 'active'",
		key,
	).Scan(&sprint.BoardKey, &sprint.Number, &sprint.Name, &sprint.Goal, &sprint.Status, &sprint.StartDate, &sprint.EndDate, &sprint.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSprintNotFound
		}
		return nil, fmt.Errorf("failed to get active sprint: %w", err)
	}
	return sprint, nil
}

func (s *Store) CreateSprint(ctx context.Context, sprint *models.Sprint) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE boards SET sprint_seq = sprint_seq + 1, updated_at = now() WHERE key = $1 RETURNING sprint_seq",
		sprint.BoardKey,
	).Scan(&sprint.Number)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBoardNotFound
		}
		return fmt.Errorf("failed to allocate sprint number: %w", err)
	}
	err = s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO sprints (board_key, number, name, goal, status, start_date, end_date, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, now()) RETURNING created_at",
		sprint.BoardKey, sprint.Number, sprint.Name, sprint.Goal, sprint.Status, sprint.StartDate, sprint.EndDate,
	).Scan(&sprint.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create sprint: %w", err)
	}
	return nil
}

func (s *Store) UpdateSprint(ctx context.Context, sprint *models.Sprint) error {
	_, err := s.querier(ctx).Exec(
		ctx,
		"UPDATE sprints SET name = $3, goal = $4, status = $5, start_date = $6, end_date = $7 WHERE board_key = $1 AND number = $2",
		sprint.BoardKey, sprint.Number, sprint.Name, sprint.Goal, sprint.Status, sprint.StartDate, sprint.EndDate,
	)
	if err != nil {
		return fmt.Errorf("failed to update sprint: %w", err)
	}
	return nil
}
