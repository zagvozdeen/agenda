package api

import "context"

func (a *API) registerActivity() {
	registerTool(a, "list_activity", "List task events by board, task or actor. Board, membership and sprint changes are not logged.", true, true, a.ListActivity)
}

func (a *API) ListActivity(ctx context.Context, input ListActivityInput) (any, error) {
	return nil, ErrNotImplemented
}
