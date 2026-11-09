package api

import "context"

func (a *API) registerSprints() {
	registerTool(a, "list_sprints", "List sprints on a board.", true, true, a.ListSprints)
	registerTool(a, "create_sprint", "Plan a sprint. Requires owner or member role on the board.", false, false, a.CreateSprint)
	registerTool(a, "update_sprint", "Update sprint name, goal or dates. Requires board membership.", false, false, a.UpdateSprint)
	registerTool(a, "start_sprint", "Activate a planned sprint. A board may have only one active sprint. Requires board membership.", false, false, a.StartSprint)
	registerTool(a, "close_sprint", "Close a sprint and move unfinished tasks to a selected planned sprint or backlog. Done and cancelled tasks stay. Requires board membership.", false, false, a.CloseSprint)
}

func (a *API) ListSprints(ctx context.Context, input ListSprintsInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) CreateSprint(ctx context.Context, input CreateSprintInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) UpdateSprint(ctx context.Context, input UpdateSprintInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) StartSprint(ctx context.Context, input SprintInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) CloseSprint(ctx context.Context, input CloseSprintInput) (any, error) {
	return nil, ErrNotImplemented
}
