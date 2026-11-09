package api

import (
	"context"
	"errors"
	"slices"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

func (a *API) lockLinkedTasks(ctx context.Context, input TaskLinkInput) error {
	if input.FromTaskID == input.ToTaskID {
		return invalid("a task cannot be linked to itself")
	}
	from, err := a.task(ctx, input.FromTaskID)
	if err != nil {
		return err
	}
	to, err := a.task(ctx, input.ToTaskID)
	if err != nil {
		return err
	}
	keys := []string{from.BoardKey, to.BoardKey}
	slices.Sort(keys)
	for _, key := range slices.Compact(keys) {
		if _, err = a.board(ctx, key, true, false); err != nil {
			return err
		}
	}
	if _, err = a.task(ctx, input.FromTaskID); err != nil {
		return err
	}
	_, err = a.task(ctx, input.ToTaskID)
	return err
}

func (a *API) LinkTasks(ctx context.Context, input TaskLinkInput) (any, error) {
	if err := a.lockLinkedTasks(ctx, input); err != nil {
		return nil, err
	}
	existing, err := a.store.GetTaskLink(ctx, input.FromTaskID, input.ToTaskID, input.Type)
	if err != nil && !errors.Is(err, store.ErrTaskLinkNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	link := &models.TaskLink{FromTaskID: input.FromTaskID, ToTaskID: input.ToTaskID, Type: input.Type, CreatedBy: actor(ctx).ID}
	if err = a.store.CreateTaskLink(ctx, link); err != nil {
		return nil, err
	}
	for _, id := range []string{link.FromTaskID, link.ToTaskID} {
		if err = a.event(ctx, id, enums.EventTypeLinked, "link", nil, link); err != nil {
			return nil, err
		}
	}
	return link, nil
}

func (a *API) UnlinkTasks(ctx context.Context, input TaskLinkInput) (any, error) {
	if err := a.lockLinkedTasks(ctx, input); err != nil {
		return nil, err
	}
	link, err := a.store.GetTaskLink(ctx, input.FromTaskID, input.ToTaskID, input.Type)
	if err != nil {
		return nil, err
	}
	if err = a.store.DeleteTaskLink(ctx, input.FromTaskID, input.ToTaskID, input.Type); err != nil {
		return nil, err
	}
	for _, id := range []string{link.FromTaskID, link.ToTaskID} {
		if err = a.event(ctx, id, enums.EventTypeUnlinked, "link", link, nil); err != nil {
			return nil, err
		}
	}
	return map[string]any{"from_task_id": link.FromTaskID, "to_task_id": link.ToTaskID, "type": link.Type, "removed": true}, nil
}
