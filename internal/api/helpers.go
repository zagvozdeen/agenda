package api

import (
	"context"
	"encoding/json/v2"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

var boardKeyPattern = regexp.MustCompile("^[A-Z]{3}$")
var taskIDPattern = regexp.MustCompile("^[A-Z]{3}-[1-9][0-9]{0,9}$")

func pagination(limit, offset int) (int, error) {
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return 0, invalid("limit must be between 1 and 100 and offset must be nonnegative")
	}
	return limit, nil
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return "", invalid("name or title must contain 1 to 255 characters")
	}
	return name, nil
}

func validTaskID(id string) error {
	if !taskIDPattern.MatchString(id) {
		return invalid("task ID must have the form ABC-12")
	}
	number, err := strconv.ParseInt(strings.SplitN(id, "-", 2)[1], 10, 32)
	if err != nil || number <= 0 {
		return invalid("invalid task number")
	}
	return nil
}

func (a *API) board(ctx context.Context, key string, write, ownerOnly bool) (*models.Board, error) {
	if !boardKeyPattern.MatchString(key) {
		return nil, invalid("board key must contain exactly three uppercase Latin letters")
	}
	board, err := a.store.GetBoard(ctx, key, write)
	if err != nil {
		return nil, err
	}
	if write {
		user := actor(ctx)
		if user == nil {
			return nil, forbidden("authentication required")
		}
		role, err := a.store.GetBoardRole(ctx, key, user.ID)
		if err != nil {
			if errors.Is(err, store.ErrBoardMemberNotFound) {
				return nil, forbidden("board membership is required")
			}
			return nil, err
		}
		if ownerOnly && role != enums.BoardRoleOwner {
			return nil, forbidden("board owner role is required")
		}
	}
	return board, nil
}

func (a *API) task(ctx context.Context, id string) (*models.Task, error) {
	if err := validTaskID(id); err != nil {
		return nil, err
	}
	return a.store.GetTask(ctx, id)
}

func (a *API) writableTask(ctx context.Context, id string) (*models.Task, error) {
	task, err := a.task(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err = a.board(ctx, task.BoardKey, true, false); err != nil {
		return nil, err
	}
	return a.store.GetTask(ctx, id)
}

func (a *API) checkAssignee(ctx context.Context, key string, userID *uuid.UUID) error {
	if userID == nil {
		return nil
	}
	if _, err := a.store.GetUserByID(ctx, *userID); err != nil {
		return err
	}
	_, err := a.store.GetBoardRole(ctx, key, *userID)
	if err != nil {
		if errors.Is(err, store.ErrBoardMemberNotFound) {
			return invalid("assignee must be a board member")
		}
		return err
	}
	return nil
}

func (a *API) checkSprint(ctx context.Context, key string, number *int) error {
	if number == nil {
		return nil
	}
	if *number <= 0 || *number > math.MaxInt32 {
		return invalid("sprint number must be a positive 32-bit integer")
	}
	sprint, err := a.store.GetSprint(ctx, key, *number)
	if err != nil {
		return err
	}
	if sprint.Status == enums.SprintStatusClosed {
		return conflict("tasks cannot be added to a closed sprint")
	}
	return nil
}

func (a *API) checkParent(ctx context.Context, key, taskID string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	parent, err := a.task(ctx, *parentID)
	if err != nil {
		return err
	}
	if parent.BoardKey != key {
		return invalid("parent task must belong to the same board")
	}
	if taskID != "" {
		cycle, err := a.store.ParentCreatesCycle(ctx, parent.ID, taskID)
		if err != nil {
			return err
		}
		if cycle {
			return conflict("parent task would create a hierarchy cycle")
		}
	}
	return nil
}

func parseDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	date, err := time.Parse(time.DateOnly, *value)
	if err != nil {
		return nil, invalid("sprint dates must use YYYY-MM-DD format")
	}
	return &date, nil
}

func validDates(start, end *time.Time) error {
	if start != nil && end != nil && end.Before(*start) {
		return invalid("sprint end date must not precede its start date")
	}
	return nil
}

func (a *API) event(ctx context.Context, taskID string, kind enums.EventType, field string, oldValue, newValue any) error {
	event := &models.TaskEvent{TaskID: taskID, ActorID: actor(ctx).ID, EventType: kind}
	if field != "" {
		event.Field = &field
	}
	var err error
	if oldValue != nil {
		event.OldValue, err = json.Marshal(oldValue)
		if err != nil {
			return err
		}
	}
	if newValue != nil {
		event.NewValue, err = json.Marshal(newValue)
		if err != nil {
			return err
		}
	}
	return a.store.CreateTaskEvent(ctx, event)
}
