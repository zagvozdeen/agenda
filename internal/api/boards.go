package api

import "context"

func (a *API) registerBoards() {
	registerTool(a, "list_boards", "List non-deleted boards. Reading is public.", true, true, a.ListBoards)
	registerTool(a, "get_board", "Get a board, members, active sprint and task counts by status.", true, true, a.GetBoard)
	registerTool(a, "create_board", "Create a board. The authenticated creator becomes its owner.", false, false, a.CreateBoard)
	registerTool(a, "update_board", "Update a board name or description. Only owners may do this.", false, false, a.UpdateBoard)
	registerTool(a, "delete_board", "Soft-delete a board. Only owners may do this.", false, false, a.DeleteBoard)
	registerTool(a, "add_member", "Add a member or change their role. Owner only; the last owner cannot be demoted.", false, false, a.AddMember)
	registerTool(a, "remove_member", "Remove a board member. Owner only; the last owner cannot be removed.", false, false, a.RemoveMember)
}

func (a *API) ListBoards(ctx context.Context, input ListBoardsInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) GetBoard(ctx context.Context, input BoardInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) CreateBoard(ctx context.Context, input CreateBoardInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) UpdateBoard(ctx context.Context, input UpdateBoardInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) DeleteBoard(ctx context.Context, input BoardInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) AddMember(ctx context.Context, input AddMemberInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) RemoveMember(ctx context.Context, input RemoveMemberInput) (any, error) {
	return nil, ErrNotImplemented
}
