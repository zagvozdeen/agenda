package api

import (
	"context"
	"strings"
)

func (a *API) registerUsers() {
	registerTool(a, "whoami", "Show the authenticated user and their board roles.", true, false, a.WhoAmI)
	registerTool(a, "find_users", "Find active users by name or username.", true, true, a.FindUsers)
}

func (a *API) WhoAmI(ctx context.Context, _ EmptyInput) (any, error) {
	user := actor(ctx)
	boards, err := a.store.ListMemberships(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"user": user, "boards": boards}, nil
}

func (a *API) FindUsers(ctx context.Context, input FindUsersInput) (any, error) {
	limit, err := pagination(input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, invalid("user search query must not be empty")
	}
	users, err := a.store.FindUsers(ctx, query, limit, input.Offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"users": users, "limit": limit, "offset": input.Offset}, nil
}
