package store

import (
	"context"
	"fmt"
	"strings"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
)

type ActivityFilter struct {
	BoardKey *string
	TaskID   *string
	ActorID  *uuid.UUID
	Limit    int
	Offset   int
}

func (s *Store) CreateTaskEvent(ctx context.Context, event *models.TaskEvent) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO task_events (task_id, actor_id, event_type, field, old_value, new_value, created_at) VALUES ($1, $2, $3, $4, $5, $6, now()) RETURNING id, created_at",
		event.TaskID, event.ActorID, event.EventType, event.Field, []byte(event.OldValue), []byte(event.NewValue),
	).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task event: %w", err)
	}
	return nil
}

func (s *Store) ListActivity(ctx context.Context, filter ActivityFilter) ([]models.TaskEvent, error) {
	where := []string{"b.deleted_at IS NULL"}
	var args []any
	add := func(expression string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(expression, len(args)))
	}
	if filter.BoardKey != nil {
		add("t.board_key = $%d", *filter.BoardKey)
	}
	if filter.TaskID != nil {
		add("e.task_id = $%d", *filter.TaskID)
	}
	if filter.ActorID != nil {
		add("e.actor_id = $%d", *filter.ActorID)
	}
	query := "SELECT e.id, e.task_id, e.actor_id, e.event_type, e.field, e.old_value, e.new_value, e.created_at FROM task_events AS e JOIN tasks AS t ON t.id = e.task_id JOIN boards AS b ON b.key = t.board_key WHERE " + strings.Join(where, " AND ") + " ORDER BY e.created_at DESC, e.id DESC"
	if filter.Limit > 0 {
		args = append(args, filter.Limit, filter.Offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}
	rows, err := s.querier(ctx).Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list task events: %w", err)
	}
	defer rows.Close()
	var events []models.TaskEvent
	for rows.Next() {
		var event models.TaskEvent
		err = rows.Scan(&event.ID, &event.TaskID, &event.ActorID, &event.EventType, &event.Field, &event.OldValue, &event.NewValue, &event.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list task events: %w", err)
		}
		events = append(events, event)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list task events: %w", err)
	}
	return events, nil
}
