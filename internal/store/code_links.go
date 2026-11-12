package store

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
)

var ErrCodeLinkNotFound = errors.New("code link not found")

const codeLinkColumns = "c.id, c.task_id, c.type, c.url, c.base_sha, c.head_sha, c.additions, c.deletions, c.created_by, c.created_at, c.updated_at"

func (s *Store) GetCodeLink(ctx context.Context, id uuid.UUID) (*models.CodeLink, error) {
	link := &models.CodeLink{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT "+codeLinkColumns+" FROM task_code_links AS c JOIN tasks AS t ON t.id = c.task_id JOIN boards AS b ON b.key = t.board_key WHERE c.id = $1 AND t.deleted_at IS NULL AND b.deleted_at IS NULL",
		id,
	).Scan(&link.ID, &link.TaskID, &link.Type, &link.URL, &link.BaseSHA, &link.HeadSHA, &link.Additions, &link.Deletions, &link.CreatedBy, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCodeLinkNotFound
		}
		return nil, fmt.Errorf("failed to get code link: %w", err)
	}
	return link, nil
}

func (s *Store) ListCodeLinks(ctx context.Context, taskID string) ([]models.CodeLink, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT "+codeLinkColumns+" FROM task_code_links AS c JOIN tasks AS t ON t.id = c.task_id JOIN boards AS b ON b.key = t.board_key WHERE c.task_id = $1 AND t.deleted_at IS NULL AND b.deleted_at IS NULL ORDER BY c.created_at, c.id",
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list code links: %w", err)
	}
	defer rows.Close()
	var links []models.CodeLink
	for rows.Next() {
		var link models.CodeLink
		err = rows.Scan(&link.ID, &link.TaskID, &link.Type, &link.URL, &link.BaseSHA, &link.HeadSHA, &link.Additions, &link.Deletions, &link.CreatedBy, &link.CreatedAt, &link.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list code links: %w", err)
		}
		links = append(links, link)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list code links: %w", err)
	}
	return links, nil
}

func (s *Store) ListCodeLinksForTasks(ctx context.Context, taskIDs []string) (map[string][]models.CodeLink, error) {
	links := make(map[string][]models.CodeLink, len(taskIDs))
	if len(taskIDs) == 0 {
		return links, nil
	}
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT "+codeLinkColumns+" FROM task_code_links AS c JOIN tasks AS t ON t.id = c.task_id JOIN boards AS b ON b.key = t.board_key WHERE c.task_id = ANY($1) AND t.deleted_at IS NULL AND b.deleted_at IS NULL ORDER BY c.created_at, c.id",
		taskIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list task code links: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var link models.CodeLink
		err = rows.Scan(&link.ID, &link.TaskID, &link.Type, &link.URL, &link.BaseSHA, &link.HeadSHA, &link.Additions, &link.Deletions, &link.CreatedBy, &link.CreatedAt, &link.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list task code links: %w", err)
		}
		links[link.TaskID] = append(links[link.TaskID], link)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list task code links: %w", err)
	}
	return links, nil
}

func (s *Store) CodeLinkSummaries(ctx context.Context, taskIDs []string) (map[string]models.CodeSummary, error) {
	summaries := make(map[string]models.CodeSummary, len(taskIDs))
	if len(taskIDs) == 0 {
		return summaries, nil
	}
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT c.task_id, count(*), count(c.additions), sum(c.additions), sum(c.deletions) FROM task_code_links AS c JOIN tasks AS t ON t.id = c.task_id JOIN boards AS b ON b.key = t.board_key WHERE c.task_id = ANY($1) AND t.deleted_at IS NULL AND b.deleted_at IS NULL GROUP BY c.task_id",
		taskIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get code link summaries: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var taskID string
		var summary models.CodeSummary
		if err = rows.Scan(&taskID, &summary.Links, &summary.WithStats, &summary.Additions, &summary.Deletions); err != nil {
			return nil, fmt.Errorf("failed to get code link summaries: %w", err)
		}
		summaries[taskID] = summary
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get code link summaries: %w", err)
	}
	return summaries, nil
}

func (s *Store) CreateCodeLink(ctx context.Context, link *models.CodeLink) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO task_code_links (task_id, type, url, base_sha, head_sha, additions, deletions, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now()) RETURNING id, created_at, updated_at",
		link.TaskID, link.Type, link.URL, link.BaseSHA, link.HeadSHA, link.Additions, link.Deletions, link.CreatedBy,
	).Scan(&link.ID, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create code link: %w", err)
	}
	return nil
}

func (s *Store) UpdateCodeLink(ctx context.Context, link *models.CodeLink) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE task_code_links SET type = $2, url = $3, base_sha = $4, head_sha = $5, additions = $6, deletions = $7, updated_at = now() WHERE id = $1 RETURNING updated_at",
		link.ID, link.Type, link.URL, link.BaseSHA, link.HeadSHA, link.Additions, link.Deletions,
	).Scan(&link.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCodeLinkNotFound
		}
		return fmt.Errorf("failed to update code link: %w", err)
	}
	return nil
}

func (s *Store) DeleteCodeLink(ctx context.Context, id uuid.UUID) error {
	_, err := s.querier(ctx).Exec(
		ctx,
		"DELETE FROM task_code_links WHERE id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete code link: %w", err)
	}
	return nil
}
