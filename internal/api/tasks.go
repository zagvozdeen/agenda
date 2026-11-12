package api

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"time"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

func (a *API) registerTasks() {
	registerTool(a, "create_task", "Create a task or subtask with an automatically assigned ID. Requires board membership.", false, false, a.CreateTask)
	registerTool(a, "get_task", "Get a task with its ordered checklist, deadline, code links and diff totals, subtasks, task links, comments and events.", true, true, a.GetTask)
	registerTool(a, "update_task", "Update mutable task fields. Board key and task number are immutable. Requires board membership.", false, false, a.UpdateTask)
	registerTool(a, "delete_task", "Soft-delete a task. Delete its non-deleted children first. Requires board membership.", false, false, a.DeleteTask)
	registerTool(a, "search_tasks", "Search current task fields with filters, full-text search and sorting. Historical assignments and sprint membership are not searched.", true, true, a.SearchTasks)
	registerTool(a, "link_tasks", "Create a directed relationship between tasks. Requires board membership.", false, false, a.LinkTasks)
	registerTool(a, "unlink_tasks", "Remove a relationship between tasks. Requires board membership.", false, false, a.UnlinkTasks)
}

func (a *API) CreateTask(ctx context.Context, input CreateTaskInput) (any, error) {
	title, err := validName(input.Title)
	if err != nil {
		return nil, err
	}
	if input.EstimateHours != nil && (*input.EstimateHours < 0 || *input.EstimateHours > math.MaxInt32) {
		return nil, invalid("estimate_hours must be a nonnegative 32-bit integer")
	}
	deadline, err := validDeadline(input.Deadline)
	if err != nil {
		return nil, err
	}
	if input.IdempotencyKey != nil {
		if *input.IdempotencyKey == uuid.Nil() {
			return nil, invalid("idempotency key must not be the nil UUID")
		}
		if err = a.store.LockIdempotency(ctx, *input.IdempotencyKey); err != nil {
			return nil, err
		}
	}
	board, err := a.board(ctx, input.BoardKey, true, false)
	if err != nil {
		return nil, err
	}
	if input.IdempotencyKey != nil {
		existing, err := a.store.GetTaskByIdempotency(ctx, *input.IdempotencyKey)
		if err != nil && !errors.Is(err, store.ErrTaskNotFound) {
			return nil, err
		}
		if existing != nil {
			if existing.BoardKey != input.BoardKey || existing.CreatedBy != actor(ctx).ID || existing.DeletedAt != nil {
				return nil, conflict("idempotency key has already been used for another task")
			}
			return existing, nil
		}
	}
	if board.TaskSeq == math.MaxInt32 {
		return nil, conflict("task numbering limit reached")
	}
	if err = a.checkAssignee(ctx, input.BoardKey, input.AssigneeID); err != nil {
		return nil, err
	}
	if err = a.checkSprint(ctx, input.BoardKey, input.SprintNumber); err != nil {
		return nil, err
	}
	if err = a.checkParent(ctx, input.BoardKey, "", input.ParentID); err != nil {
		return nil, err
	}
	if input.Type == "" {
		input.Type = enums.TaskTypeFeature
	}
	if input.Priority == "" {
		input.Priority = enums.TaskPriorityMedium
	}
	task := &models.Task{
		BoardKey: input.BoardKey, Title: title, Description: input.Description,
		Type: input.Type, Status: enums.TaskStatusTodo, Priority: input.Priority,
		AssigneeID: input.AssigneeID, CreatedBy: actor(ctx).ID, SprintNumber: input.SprintNumber,
		ParentID: input.ParentID, EstimateHours: input.EstimateHours, IdempotencyKey: input.IdempotencyKey,
		Deadline: deadline,
	}
	if err = a.store.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	if err = a.event(ctx, task.ID, enums.EventTypeCreated, "", nil, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (a *API) GetTask(ctx context.Context, input TaskInput) (any, error) {
	task, err := a.task(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}
	view := &models.TaskView{Task: *task}
	view.Checklist, err = a.store.GetChecklist(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	view.CodeLinks, err = a.store.ListCodeLinks(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	summaries, err := a.store.CodeLinkSummaries(ctx, []string{task.ID})
	if err != nil {
		return nil, err
	}
	view.CodeChanges = summaries[task.ID]
	view.Subtasks, err = a.store.ListSubtasks(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	view.Links, err = a.store.ListTaskLinks(ctx, task.ID, viewerID(ctx))
	if err != nil {
		return nil, err
	}
	view.Comments, err = a.store.ListComments(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	view.Events, err = a.store.ListActivity(ctx, store.ActivityFilter{TaskID: &task.ID, ViewerID: viewerID(ctx)})
	return view, err
}

func (a *API) UpdateTask(ctx context.Context, input UpdateTaskInput) (any, error) {
	task, err := a.writableTask(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}
	if input.Title == nil && input.Description == nil && input.Type == nil && input.Status == nil && input.Priority == nil && !input.AssigneeID.Set && !input.SprintNumber.Set && !input.ParentID.Set && !input.EstimateHours.Set && !input.Deadline.Set {
		return nil, invalid("at least one task field must be provided")
	}
	before := *task
	if input.Deadline.Set {
		deadline, err := validDeadline(input.Deadline.Value)
		if err != nil {
			return nil, err
		}
		if deadline == nil || task.Deadline == nil || !deadline.Equal(*task.Deadline) {
			task.Deadline = deadline
		}
	}
	if input.Title != nil {
		task.Title, err = validName(*input.Title)
		if err != nil {
			return nil, err
		}
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Type != nil {
		task.Type = *input.Type
	}
	if input.Status != nil {
		task.Status = *input.Status
	}
	if input.Priority != nil {
		task.Priority = *input.Priority
	}
	if input.AssigneeID.Set {
		if err = a.checkAssignee(ctx, task.BoardKey, input.AssigneeID.Value); err != nil {
			return nil, err
		}
		task.AssigneeID = input.AssigneeID.Value
	}
	if input.SprintNumber.Set && !reflect.DeepEqual(task.SprintNumber, input.SprintNumber.Value) {
		if err = a.checkSprint(ctx, task.BoardKey, input.SprintNumber.Value); err != nil {
			return nil, err
		}
		task.SprintNumber = input.SprintNumber.Value
	}
	if input.ParentID.Set {
		if err = a.checkParent(ctx, task.BoardKey, task.ID, input.ParentID.Value); err != nil {
			return nil, err
		}
		task.ParentID = input.ParentID.Value
	}
	if input.EstimateHours.Set {
		if input.EstimateHours.Value != nil && (*input.EstimateHours.Value < 0 || *input.EstimateHours.Value > math.MaxInt32) {
			return nil, invalid("estimate_hours must be a nonnegative 32-bit integer")
		}
		task.EstimateHours = input.EstimateHours.Value
	}
	if task.Status != before.Status {
		if task.Status == enums.TaskStatusDone {
			now := time.Now().UTC()
			task.CompletedAt = &now
		} else {
			task.CompletedAt = nil
		}
	}
	if reflect.DeepEqual(before, *task) {
		return task, nil
	}
	if err = a.store.UpdateTask(ctx, task); err != nil {
		return nil, err
	}
	changes := []struct {
		field      string
		old, value any
	}{
		{"title", before.Title, task.Title}, {"description", before.Description, task.Description},
		{"type", before.Type, task.Type}, {"status", before.Status, task.Status},
		{"priority", before.Priority, task.Priority}, {"assignee_id", before.AssigneeID, task.AssigneeID},
		{"sprint_number", before.SprintNumber, task.SprintNumber}, {"parent_id", before.ParentID, task.ParentID},
		{"estimate_hours", before.EstimateHours, task.EstimateHours}, {"completed_at", before.CompletedAt, task.CompletedAt},
		{"deadline", before.Deadline, task.Deadline},
	}
	for _, change := range changes {
		if !reflect.DeepEqual(change.old, change.value) {
			if err = a.event(ctx, task.ID, enums.EventTypeUpdated, change.field, change.old, change.value); err != nil {
				return nil, err
			}
		}
	}
	return task, nil
}

func (a *API) DeleteTask(ctx context.Context, input TaskInput) (any, error) {
	task, err := a.writableTask(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}
	hasChildren, err := a.store.HasTaskChildren(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if hasChildren {
		return nil, conflict("delete non-deleted subtasks before deleting their parent")
	}
	before := *task
	if err = a.store.DeleteTask(ctx, task); err != nil {
		return nil, err
	}
	if err = a.event(ctx, task.ID, enums.EventTypeDeleted, "", before, nil); err != nil {
		return nil, err
	}
	return task, nil
}

func (a *API) SearchTasks(ctx context.Context, input SearchTasksInput) (any, error) {
	limit, err := pagination(input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}
	if input.BoardKey != nil {
		if _, err = a.board(ctx, *input.BoardKey, false, false); err != nil {
			return nil, err
		}
	}
	if input.SprintNumber != nil {
		if input.BoardKey == nil || *input.SprintNumber <= 0 || *input.SprintNumber > math.MaxInt32 {
			return nil, invalid("a positive sprint number requires board_key")
		}
		if input.Backlog {
			return nil, invalid("backlog and sprint_number cannot be combined")
		}
	}
	if input.Unassigned && input.AssigneeID != nil {
		return nil, invalid("unassigned and assignee_id cannot be combined")
	}
	if input.ParentID != nil {
		if err = validTaskID(*input.ParentID); err != nil {
			return nil, err
		}
	}
	if input.Query != nil {
		query := strings.TrimSpace(*input.Query)
		if query == "" {
			return nil, invalid("task search query must not be empty")
		}
		input.Query = &query
	}
	input.Sort = strings.ToLower(input.Sort)
	input.Direction = strings.ToLower(input.Direction)
	if input.Sort == "" {
		input.Sort = "updated_at"
	}
	if input.Direction == "" {
		input.Direction = "desc"
	}
	switch input.Sort {
	case "created_at", "updated_at", "priority", "number", "completed_at":
	default:
		return nil, invalid("unsupported task sort field")
	}
	if input.Direction != "asc" && input.Direction != "desc" {
		return nil, invalid("sort direction must be asc or desc")
	}
	tasks, err := a.store.SearchTasks(ctx, store.TaskFilter{
		ViewerID: viewerID(ctx),
		BoardKey: input.BoardKey, Query: input.Query, AssigneeID: input.AssigneeID, CreatedBy: input.CreatedBy,
		SprintNumber: input.SprintNumber, ParentID: input.ParentID, Type: input.Type, Status: input.Status, Priority: input.Priority,
		Backlog: input.Backlog, Unassigned: input.Unassigned, Sort: input.Sort, Direction: input.Direction, Limit: limit, Offset: input.Offset,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(tasks))
	for index, task := range tasks {
		ids[index] = task.ID
	}
	summaries, err := a.store.CodeLinkSummaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make([]models.TaskSummary, len(tasks))
	for index, task := range tasks {
		items[index] = models.TaskSummary{Task: task, CodeChanges: summaries[task.ID]}
	}
	return map[string]any{"tasks": items, "limit": limit, "offset": input.Offset}, nil
}
