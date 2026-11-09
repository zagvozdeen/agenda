package api

import (
	"context"
	"errors"
	"math"
	"reflect"
	"unicode/utf8"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

func (a *API) registerSprints() {
	registerTool(a, "list_sprints", "List sprints on a board.", true, true, a.ListSprints)
	registerTool(a, "create_sprint", "Plan a sprint. Requires owner or member role on the board.", false, false, a.CreateSprint)
	registerTool(a, "update_sprint", "Update sprint name, goal or dates. Requires board membership.", false, false, a.UpdateSprint)
	registerTool(a, "start_sprint", "Activate a planned sprint. A board may have only one active sprint. Requires board membership.", false, false, a.StartSprint)
	registerTool(a, "close_sprint", "Close a sprint and move unfinished tasks to a selected planned sprint or backlog. Done and cancelled tasks stay. Requires board membership.", false, false, a.CloseSprint)
}

func (a *API) ListSprints(ctx context.Context, input ListSprintsInput) (any, error) {
	limit, err := pagination(input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}
	if _, err = a.board(ctx, input.BoardKey, false, false); err != nil {
		return nil, err
	}
	sprints, err := a.store.ListSprints(ctx, input.BoardKey, input.Status, limit, input.Offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"sprints": sprints, "limit": limit, "offset": input.Offset}, nil
}

func (a *API) CreateSprint(ctx context.Context, input CreateSprintInput) (any, error) {
	board, err := a.board(ctx, input.BoardKey, true, false)
	if err != nil {
		return nil, err
	}
	if board.SprintSeq == math.MaxInt32 {
		return nil, conflict("sprint numbering limit reached")
	}
	if input.Name != nil && utf8.RuneCountInString(*input.Name) > 255 {
		return nil, invalid("sprint name must not exceed 255 characters")
	}
	start, err := parseDate(input.StartDate)
	if err != nil {
		return nil, err
	}
	end, err := parseDate(input.EndDate)
	if err != nil {
		return nil, err
	}
	if err = validDates(start, end); err != nil {
		return nil, err
	}
	sprint := &models.Sprint{BoardKey: input.BoardKey, Name: input.Name, Goal: input.Goal, Status: enums.SprintStatusPlanned, StartDate: start, EndDate: end}
	if err = a.store.CreateSprint(ctx, sprint); err != nil {
		return nil, err
	}
	return sprint, nil
}

func (a *API) UpdateSprint(ctx context.Context, input UpdateSprintInput) (any, error) {
	if _, err := a.board(ctx, input.BoardKey, true, false); err != nil {
		return nil, err
	}
	if input.Number <= 0 || input.Number > math.MaxInt32 {
		return nil, invalid("sprint number must be a positive 32-bit integer")
	}
	if !input.Name.Set && !input.Goal.Set && !input.StartDate.Set && !input.EndDate.Set {
		return nil, invalid("at least one sprint field must be provided")
	}
	sprint, err := a.store.GetSprint(ctx, input.BoardKey, input.Number)
	if err != nil {
		return nil, err
	}
	before := *sprint
	if input.Name.Set {
		if input.Name.Value != nil && utf8.RuneCountInString(*input.Name.Value) > 255 {
			return nil, invalid("sprint name must not exceed 255 characters")
		}
		sprint.Name = input.Name.Value
	}
	if input.Goal.Set {
		sprint.Goal = input.Goal.Value
	}
	if input.StartDate.Set {
		sprint.StartDate, err = parseDate(input.StartDate.Value)
		if err != nil {
			return nil, err
		}
	}
	if input.EndDate.Set {
		sprint.EndDate, err = parseDate(input.EndDate.Value)
		if err != nil {
			return nil, err
		}
	}
	if err = validDates(sprint.StartDate, sprint.EndDate); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(before, *sprint) {
		if err = a.store.UpdateSprint(ctx, sprint); err != nil {
			return nil, err
		}
	}
	return sprint, nil
}

func (a *API) StartSprint(ctx context.Context, input SprintInput) (any, error) {
	if _, err := a.board(ctx, input.BoardKey, true, false); err != nil {
		return nil, err
	}
	if input.Number <= 0 || input.Number > math.MaxInt32 {
		return nil, invalid("sprint number must be a positive 32-bit integer")
	}
	sprint, err := a.store.GetSprint(ctx, input.BoardKey, input.Number)
	if err != nil {
		return nil, err
	}
	if sprint.Status == enums.SprintStatusActive {
		return sprint, nil
	}
	if sprint.Status != enums.SprintStatusPlanned {
		return nil, conflict("only a planned sprint can be started")
	}
	active, err := a.store.GetActiveSprint(ctx, input.BoardKey)
	if err != nil && !errors.Is(err, store.ErrSprintNotFound) {
		return nil, err
	}
	if active != nil {
		return nil, conflict("the board already has an active sprint")
	}
	sprint.Status = enums.SprintStatusActive
	if err = a.store.UpdateSprint(ctx, sprint); err != nil {
		return nil, err
	}
	return sprint, nil
}

func (a *API) CloseSprint(ctx context.Context, input CloseSprintInput) (any, error) {
	if _, err := a.board(ctx, input.BoardKey, true, false); err != nil {
		return nil, err
	}
	if input.Number <= 0 || input.Number > math.MaxInt32 {
		return nil, invalid("sprint number must be a positive 32-bit integer")
	}
	sprint, err := a.store.GetSprint(ctx, input.BoardKey, input.Number)
	if err != nil {
		return nil, err
	}
	if sprint.Status != enums.SprintStatusActive {
		return nil, conflict("only an active sprint can be closed")
	}
	if input.TargetSprintNumber != nil {
		if *input.TargetSprintNumber <= 0 || *input.TargetSprintNumber > math.MaxInt32 || *input.TargetSprintNumber == input.Number {
			return nil, invalid("target sprint must be a different planned sprint")
		}
		target, err := a.store.GetSprint(ctx, input.BoardKey, *input.TargetSprintNumber)
		if err != nil {
			return nil, err
		}
		if target.Status != enums.SprintStatusPlanned {
			return nil, conflict("unfinished tasks can only be moved to a planned sprint")
		}
	}
	tasks, err := a.store.ListUnfinishedTasks(ctx, input.BoardKey, input.Number)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		task := &tasks[i]
		old := task.SprintNumber
		task.SprintNumber = input.TargetSprintNumber
		if err = a.store.UpdateTask(ctx, task); err != nil {
			return nil, err
		}
		if err = a.event(ctx, task.ID, enums.EventTypeUpdated, "sprint_number", old, task.SprintNumber); err != nil {
			return nil, err
		}
	}
	sprint.Status = enums.SprintStatusClosed
	if err = a.store.UpdateSprint(ctx, sprint); err != nil {
		return nil, err
	}
	return map[string]any{"sprint": sprint, "moved_tasks": len(tasks), "target_sprint_number": input.TargetSprintNumber}, nil
}
