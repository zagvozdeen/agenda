package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

var ErrTaskNotFound = errors.New("task not found")

const taskColumns = "t.id, t.board_key, t.number, t.title, t.description, t.type, t.status, t.priority, t.assignee_id, t.created_by, t.sprint_number, t.parent_id, t.estimate_hours, t.idempotency_key, t.created_at, t.updated_at, t.completed_at, t.deleted_at, t.deadline"

type TaskFilter struct {
	BoardKey     *string
	Query        *string
	AssigneeID   *uuid.UUID
	CreatedBy    *uuid.UUID
	SprintNumber *int
	ParentID     *string
	Type         *enums.TaskType
	Status       *enums.TaskStatus
	Priority     *enums.TaskPriority
	Backlog      bool
	Unassigned   bool
	Sort         string
	Direction    string
	Limit        int
	Offset       int
}

func (s *Store) GetTask(ctx context.Context, id string) (*models.Task, error) {
	task := &models.Task{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT "+taskColumns+" FROM tasks AS t JOIN boards AS b ON b.key = t.board_key WHERE t.id = $1 AND t.deleted_at IS NULL AND b.deleted_at IS NULL",
		id,
	).Scan(&task.ID, &task.BoardKey, &task.Number, &task.Title, &task.Description, &task.Type, &task.Status, &task.Priority, &task.AssigneeID, &task.CreatedBy, &task.SprintNumber, &task.ParentID, &task.EstimateHours, &task.IdempotencyKey, &task.CreatedAt, &task.UpdatedAt, &task.CompletedAt, &task.DeletedAt, &task.Deadline)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

func (s *Store) TaskExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM tasks AS t JOIN boards AS b ON b.key = t.board_key WHERE t.id = $1 AND b.deleted_at IS NULL)",
		id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check task existence: %w", err)
	}
	return exists, nil
}

func (s *Store) LockIdempotency(ctx context.Context, key uuid.UUID) error {
	_, err := s.querier(ctx).Exec(
		ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		key.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to lock idempotency key: %w", err)
	}
	return nil
}

func (s *Store) GetTaskByIdempotency(ctx context.Context, key uuid.UUID) (*models.Task, error) {
	task := &models.Task{}
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT "+taskColumns+" FROM tasks AS t WHERE t.idempotency_key = $1",
		key,
	).Scan(&task.ID, &task.BoardKey, &task.Number, &task.Title, &task.Description, &task.Type, &task.Status, &task.Priority, &task.AssigneeID, &task.CreatedBy, &task.SprintNumber, &task.ParentID, &task.EstimateHours, &task.IdempotencyKey, &task.CreatedAt, &task.UpdatedAt, &task.CompletedAt, &task.DeletedAt, &task.Deadline)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task by idempotency key: %w", err)
	}
	return task, nil
}

func (s *Store) CreateTask(ctx context.Context, task *models.Task) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE boards SET task_seq = task_seq + 1, updated_at = now() WHERE key = $1 RETURNING task_seq",
		task.BoardKey,
	).Scan(&task.Number)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBoardNotFound
		}
		return fmt.Errorf("failed to allocate task number: %w", err)
	}
	err = s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO tasks (board_key, number, title, description, type, status, priority, assignee_id, created_by, sprint_number, parent_id, estimate_hours, idempotency_key, deadline, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, now(), now()) RETURNING id, created_at, updated_at",
		task.BoardKey, task.Number, task.Title, task.Description, task.Type, task.Status, task.Priority, task.AssigneeID, task.CreatedBy, task.SprintNumber, task.ParentID, task.EstimateHours, task.IdempotencyKey, task.Deadline,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

func (s *Store) UpdateTask(ctx context.Context, task *models.Task) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE tasks SET title = $2, description = $3, type = $4, status = $5, priority = $6, assignee_id = $7, sprint_number = $8, parent_id = $9, estimate_hours = $10, completed_at = $11, deadline = $12, updated_at = now() WHERE id = $1 RETURNING updated_at",
		task.ID, task.Title, task.Description, task.Type, task.Status, task.Priority, task.AssigneeID, task.SprintNumber, task.ParentID, task.EstimateHours, task.CompletedAt, task.Deadline,
	).Scan(&task.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

func (s *Store) DeleteTask(ctx context.Context, task *models.Task) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE tasks SET deleted_at = now(), updated_at = now() WHERE id = $1 RETURNING deleted_at, updated_at",
		task.ID,
	).Scan(&task.DeletedAt, &task.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

func (s *Store) HasTaskChildren(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM tasks WHERE parent_id = $1 AND deleted_at IS NULL)",
		id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check task children: %w", err)
	}
	return exists, nil
}

func (s *Store) ParentCreatesCycle(ctx context.Context, parentID, taskID string) (bool, error) {
	var cycle bool
	err := s.querier(ctx).QueryRow(
		ctx,
		"WITH RECURSIVE ancestors AS (SELECT id, parent_id FROM tasks WHERE id = $1 UNION SELECT t.id, t.parent_id FROM tasks AS t JOIN ancestors AS a ON t.id = a.parent_id) SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)",
		parentID, taskID,
	).Scan(&cycle)
	if err != nil {
		return false, fmt.Errorf("failed to check parent task cycle: %w", err)
	}
	return cycle, nil
}

func (s *Store) ListSubtasks(ctx context.Context, id string) ([]models.Task, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT "+taskColumns+" FROM tasks AS t WHERE t.parent_id = $1 AND t.deleted_at IS NULL ORDER BY t.number",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list subtasks: %w", err)
	}
	defer rows.Close()
	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		err = rows.Scan(&task.ID, &task.BoardKey, &task.Number, &task.Title, &task.Description, &task.Type, &task.Status, &task.Priority, &task.AssigneeID, &task.CreatedBy, &task.SprintNumber, &task.ParentID, &task.EstimateHours, &task.IdempotencyKey, &task.CreatedAt, &task.UpdatedAt, &task.CompletedAt, &task.DeletedAt, &task.Deadline)
		if err != nil {
			return nil, fmt.Errorf("failed to list subtasks: %w", err)
		}
		tasks = append(tasks, task)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list subtasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) ListUnfinishedTasks(ctx context.Context, key string, number int) ([]models.Task, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT "+taskColumns+" FROM tasks AS t WHERE t.board_key = $1 AND t.sprint_number = $2 AND t.status NOT IN ('done', 'cancelled') AND t.deleted_at IS NULL ORDER BY t.number",
		key, number,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list unfinished tasks: %w", err)
	}
	defer rows.Close()
	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		err = rows.Scan(&task.ID, &task.BoardKey, &task.Number, &task.Title, &task.Description, &task.Type, &task.Status, &task.Priority, &task.AssigneeID, &task.CreatedBy, &task.SprintNumber, &task.ParentID, &task.EstimateHours, &task.IdempotencyKey, &task.CreatedAt, &task.UpdatedAt, &task.CompletedAt, &task.DeletedAt, &task.Deadline)
		if err != nil {
			return nil, fmt.Errorf("failed to list unfinished tasks: %w", err)
		}
		tasks = append(tasks, task)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list unfinished tasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) SearchTasks(ctx context.Context, filter TaskFilter) ([]models.Task, error) {
	where := []string{"t.deleted_at IS NULL", "b.deleted_at IS NULL"}
	args := []any{}
	add := func(expression string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(expression, len(args)))
	}
	if filter.BoardKey != nil {
		add("t.board_key = $%d", *filter.BoardKey)
	}
	if filter.Query != nil {
		add("to_tsvector('russian', t.title || ' ' || t.description) @@ websearch_to_tsquery('russian', $%d)", *filter.Query)
	}
	if filter.AssigneeID != nil {
		add("t.assignee_id = $%d", *filter.AssigneeID)
	}
	if filter.CreatedBy != nil {
		add("t.created_by = $%d", *filter.CreatedBy)
	}
	if filter.SprintNumber != nil {
		add("t.sprint_number = $%d", *filter.SprintNumber)
	}
	if filter.ParentID != nil {
		add("t.parent_id = $%d", *filter.ParentID)
	}
	if filter.Type != nil {
		add("t.type = $%d", *filter.Type)
	}
	if filter.Status != nil {
		add("t.status = $%d", *filter.Status)
	}
	if filter.Priority != nil {
		add("t.priority = $%d", *filter.Priority)
	}
	if filter.Backlog {
		where = append(where, "t.sprint_number IS NULL")
	}
	if filter.Unassigned {
		where = append(where, "t.assignee_id IS NULL")
	}
	columns := map[string]string{"created_at": "t.created_at", "updated_at": "t.updated_at", "priority": "t.priority", "number": "t.number", "completed_at": "t.completed_at"}
	column, ok := columns[filter.Sort]
	if !ok || (filter.Direction != "asc" && filter.Direction != "desc") {
		return nil, fmt.Errorf("invalid task sort")
	}
	args = append(args, filter.Limit, filter.Offset)
	query := "SELECT " + taskColumns + " FROM tasks AS t JOIN boards AS b ON b.key = t.board_key WHERE " + strings.Join(where, " AND ") + " ORDER BY " + column + " " + strings.ToUpper(filter.Direction) + " NULLS LAST, t.id ASC LIMIT " + fmt.Sprintf("$%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.querier(ctx).Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}
	defer rows.Close()
	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		err = rows.Scan(&task.ID, &task.BoardKey, &task.Number, &task.Title, &task.Description, &task.Type, &task.Status, &task.Priority, &task.AssigneeID, &task.CreatedBy, &task.SprintNumber, &task.ParentID, &task.EstimateHours, &task.IdempotencyKey, &task.CreatedAt, &task.UpdatedAt, &task.CompletedAt, &task.DeletedAt, &task.Deadline)
		if err != nil {
			return nil, fmt.Errorf("failed to search tasks: %w", err)
		}
		tasks = append(tasks, task)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}
	return tasks, nil
}
