package api

import "context"

func (a *API) registerUsers() {
	registerTool(a, "whoami", "Show the authenticated user and their board roles.", true, false, a.WhoAmI)
	registerTool(a, "find_users", "Find active users by name or username.", true, true, a.FindUsers)
}

func (a *API) WhoAmI(ctx context.Context, input EmptyInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) FindUsers(ctx context.Context, input FindUsersInput) (any, error) {
	return nil, ErrNotImplemented
}
