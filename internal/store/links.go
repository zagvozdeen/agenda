package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

var ErrTaskLinkNotFound = errors.New("task link not found")

func (s *Store) GetTaskLink(ctx context.Context, from, to string, kind enums.LinkType) (*models.TaskLink, error) {
	link := &models.TaskLink{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT from_task_id, to_task_id, type, created_by, created_at FROM task_links WHERE from_task_id = $1 AND to_task_id = $2 AND type = $3",
		from, to, kind,
	).Scan(&link.FromTaskID, &link.ToTaskID, &link.Type, &link.CreatedBy, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskLinkNotFound
		}
		return nil, fmt.Errorf("failed to get task link: %w", err)
	}
	return link, nil
}

func (s *Store) CreateTaskLink(ctx context.Context, link *models.TaskLink) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO task_links (from_task_id, to_task_id, type, created_by, created_at) VALUES ($1, $2, $3, $4, now()) RETURNING created_at",
		link.FromTaskID, link.ToTaskID, link.Type, link.CreatedBy,
	).Scan(&link.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task link: %w", err)
	}
	return nil
}

func (s *Store) DeleteTaskLink(ctx context.Context, from, to string, kind enums.LinkType) error {
	_, err := s.querier(ctx).Exec(
		ctx,
		"DELETE FROM task_links WHERE from_task_id = $1 AND to_task_id = $2 AND type = $3",
		from, to, kind,
	)
	if err != nil {
		return fmt.Errorf("failed to delete task link: %w", err)
	}
	return nil
}

func (s *Store) ListTaskLinks(ctx context.Context, id string) ([]models.TaskLink, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT l.from_task_id, l.to_task_id, l.type, l.created_by, l.created_at FROM task_links AS l JOIN tasks AS f ON f.id = l.from_task_id JOIN tasks AS t ON t.id = l.to_task_id JOIN boards AS fb ON fb.key = f.board_key JOIN boards AS tb ON tb.key = t.board_key WHERE (l.from_task_id = $1 OR l.to_task_id = $1) AND f.deleted_at IS NULL AND t.deleted_at IS NULL AND fb.deleted_at IS NULL AND tb.deleted_at IS NULL ORDER BY l.created_at, l.from_task_id, l.to_task_id, l.type",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list task links: %w", err)
	}
	defer rows.Close()
	var links []models.TaskLink
	for rows.Next() {
		var link models.TaskLink
		err = rows.Scan(&link.FromTaskID, &link.ToTaskID, &link.Type, &link.CreatedBy, &link.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list task links: %w", err)
		}
		links = append(links, link)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list task links: %w", err)
	}
	return links, nil
}
