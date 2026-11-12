package store

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
)

func (s *Store) GetChecklist(ctx context.Context, taskID string) ([]models.ChecklistItem, error) {
	var data []byte
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT checklist FROM tasks WHERE id = $1 AND deleted_at IS NULL",
		taskID,
	).Scan(&data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task checklist: %w", err)
	}
	var items []models.ChecklistItem
	if err = json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to decode task checklist: %w", err)
	}
	return items, nil
}

func (s *Store) UpdateChecklist(ctx context.Context, taskID string, items []models.ChecklistItem) (time.Time, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to encode task checklist: %w", err)
	}
	var updatedAt time.Time
	err = s.querier(ctx).QueryRow(
		ctx,
		"UPDATE tasks SET checklist = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL RETURNING updated_at",
		taskID, data,
	).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrTaskNotFound
		}
		return time.Time{}, fmt.Errorf("failed to update task checklist: %w", err)
	}
	return updatedAt, nil
}
