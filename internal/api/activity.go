package api

import (
	"context"
	"strings"

	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

func (a *API) registerActivity() {
	registerTool(a, "list_activity", "List task events by board, task or actor. Board, membership and sprint changes are not logged.", true, true, a.ListActivity)
}

func (a *API) ListActivity(ctx context.Context, input ListActivityInput) (any, error) {
	limit, err := pagination(input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}
	if input.BoardKey != nil {
		if _, err = a.board(ctx, *input.BoardKey, false, false); err != nil {
			return nil, err
		}
	}
	if input.TaskID != nil {
		if err = validTaskID(*input.TaskID); err != nil {
			return nil, err
		}
		if _, err = a.board(ctx, strings.SplitN(*input.TaskID, "-", 2)[0], false, false); err != nil {
			return nil, err
		}
		if input.BoardKey != nil && strings.SplitN(*input.TaskID, "-", 2)[0] != *input.BoardKey {
			return nil, invalid("task does not belong to the selected board")
		}
		exists, err := a.store.TaskExists(ctx, *input.TaskID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, store.ErrTaskNotFound
		}
	}
	events, err := a.store.ListActivity(ctx, store.ActivityFilter{ViewerID: viewerID(ctx), BoardKey: input.BoardKey, TaskID: input.TaskID, ActorID: input.ActorID, Limit: limit, Offset: input.Offset})
	if err != nil {
		return nil, err
	}
	return map[string]any{"events": events, "limit": limit, "offset": input.Offset}, nil
}
