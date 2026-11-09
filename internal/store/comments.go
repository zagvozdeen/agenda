package store

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
)

var ErrCommentNotFound = errors.New("comment not found")

func (s *Store) GetComment(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	comment := &models.Comment{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT c.id, c.task_id, c.author_id, c.body, c.created_at, c.updated_at, c.deleted_at FROM comments AS c JOIN tasks AS t ON t.id = c.task_id JOIN boards AS b ON b.key = t.board_key WHERE c.id = $1 AND c.deleted_at IS NULL AND t.deleted_at IS NULL AND b.deleted_at IS NULL",
		id,
	).Scan(&comment.ID, &comment.TaskID, &comment.AuthorID, &comment.Body, &comment.CreatedAt, &comment.UpdatedAt, &comment.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}
	return comment, nil
}

func (s *Store) ListComments(ctx context.Context, taskID string) ([]models.Comment, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT id, task_id, author_id, body, created_at, updated_at, deleted_at FROM comments WHERE task_id = $1 AND deleted_at IS NULL ORDER BY created_at, id",
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}
	defer rows.Close()
	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		err = rows.Scan(&comment.ID, &comment.TaskID, &comment.AuthorID, &comment.Body, &comment.CreatedAt, &comment.UpdatedAt, &comment.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list comments: %w", err)
		}
		comments = append(comments, comment)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}
	return comments, nil
}

func (s *Store) CreateComment(ctx context.Context, comment *models.Comment) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO comments (task_id, author_id, body, created_at, updated_at) VALUES ($1, $2, $3, now(), now()) RETURNING id, created_at, updated_at",
		comment.TaskID, comment.AuthorID, comment.Body,
	).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}
	return nil
}

func (s *Store) UpdateComment(ctx context.Context, comment *models.Comment) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE comments SET body = $2, updated_at = now() WHERE id = $1 RETURNING updated_at",
		comment.ID, comment.Body,
	).Scan(&comment.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCommentNotFound
		}
		return fmt.Errorf("failed to update comment: %w", err)
	}
	return nil
}

func (s *Store) DeleteComment(ctx context.Context, comment *models.Comment) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE comments SET deleted_at = now(), updated_at = now() WHERE id = $1 RETURNING deleted_at, updated_at",
		comment.ID,
	).Scan(&comment.DeletedAt, &comment.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCommentNotFound
		}
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	return nil
}
