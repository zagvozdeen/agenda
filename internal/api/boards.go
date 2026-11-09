package api

import (
	"context"
	"errors"
	"reflect"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

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
	limit, err := pagination(input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}
	boards, err := a.store.ListBoards(ctx, input.MemberID, limit, input.Offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"boards": boards, "limit": limit, "offset": input.Offset}, nil
}

func (a *API) GetBoard(ctx context.Context, input BoardInput) (any, error) {
	board, err := a.board(ctx, input.BoardKey, false, false)
	if err != nil {
		return nil, err
	}
	view := &models.BoardView{Board: *board}
	view.Members, err = a.store.ListMembers(ctx, board.Key)
	if err != nil {
		return nil, err
	}
	view.ActiveSprint, err = a.store.GetActiveSprint(ctx, board.Key)
	if err != nil && !errors.Is(err, store.ErrSprintNotFound) {
		return nil, err
	}
	view.TaskCounts, err = a.store.CountTasksByStatus(ctx, board.Key)
	return view, err
}

func (a *API) CreateBoard(ctx context.Context, input CreateBoardInput) (any, error) {
	if !boardKeyPattern.MatchString(input.BoardKey) {
		return nil, invalid("board key must contain exactly three uppercase Latin letters")
	}
	name, err := validName(input.Name)
	if err != nil {
		return nil, err
	}
	board := &models.Board{Key: input.BoardKey, Name: name, Description: input.Description, CreatedBy: actor(ctx).ID}
	if err = a.store.CreateBoard(ctx, board); err != nil {
		return nil, err
	}
	member := &models.BoardMember{BoardKey: board.Key, UserID: board.CreatedBy, Role: enums.BoardRoleOwner}
	if err = a.store.SaveMember(ctx, member); err != nil {
		return nil, err
	}
	return board, nil
}

func (a *API) UpdateBoard(ctx context.Context, input UpdateBoardInput) (any, error) {
	board, err := a.board(ctx, input.BoardKey, true, true)
	if err != nil {
		return nil, err
	}
	if input.Name == nil && !input.Description.Set {
		return nil, invalid("at least one board field must be provided")
	}
	before := *board
	if input.Name != nil {
		board.Name, err = validName(*input.Name)
		if err != nil {
			return nil, err
		}
	}
	if input.Description.Set {
		board.Description = input.Description.Value
	}
	if !reflect.DeepEqual(before, *board) {
		if err = a.store.UpdateBoard(ctx, board); err != nil {
			return nil, err
		}
	}
	return board, nil
}

func (a *API) DeleteBoard(ctx context.Context, input BoardInput) (any, error) {
	if _, err := a.board(ctx, input.BoardKey, true, true); err != nil {
		return nil, err
	}
	if err := a.store.DeleteBoard(ctx, input.BoardKey); err != nil {
		return nil, err
	}
	return map[string]any{"board_key": input.BoardKey, "deleted": true}, nil
}

func (a *API) AddMember(ctx context.Context, input AddMemberInput) (any, error) {
	if _, err := a.board(ctx, input.BoardKey, true, true); err != nil {
		return nil, err
	}
	if _, err := a.store.GetUserByID(ctx, input.UserID); err != nil {
		return nil, err
	}
	role, err := a.store.GetBoardRole(ctx, input.BoardKey, input.UserID)
	if err != nil && !errors.Is(err, store.ErrBoardMemberNotFound) {
		return nil, err
	}
	if role == enums.BoardRoleOwner && input.Role != enums.BoardRoleOwner {
		count, err := a.store.CountOwners(ctx, input.BoardKey)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, conflict("the last board owner cannot be demoted")
		}
	}
	member := &models.BoardMember{BoardKey: input.BoardKey, UserID: input.UserID, Role: input.Role}
	if err = a.store.SaveMember(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (a *API) RemoveMember(ctx context.Context, input RemoveMemberInput) (any, error) {
	if _, err := a.board(ctx, input.BoardKey, true, true); err != nil {
		return nil, err
	}
	role, err := a.store.GetBoardRole(ctx, input.BoardKey, input.UserID)
	if err != nil {
		return nil, err
	}
	if role == enums.BoardRoleOwner {
		count, err := a.store.CountOwners(ctx, input.BoardKey)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, conflict("the last board owner cannot be removed")
		}
	}
	if err = a.store.RemoveMember(ctx, input.BoardKey, input.UserID); err != nil {
		return nil, err
	}
	return map[string]any{"board_key": input.BoardKey, "user_id": input.UserID, "removed": true}, nil
}
