package api

import "context"

func (a *API) registerComments() {
	registerTool(a, "add_comment", "Add a task comment. Requires board membership.", false, false, a.AddComment)
	registerTool(a, "update_comment", "Update your own comment. Requires board membership.", false, false, a.UpdateComment)
	registerTool(a, "delete_comment", "Soft-delete your own comment. Requires board membership.", false, false, a.DeleteComment)
}

func (a *API) AddComment(ctx context.Context, input AddCommentInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) UpdateComment(ctx context.Context, input UpdateCommentInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) DeleteComment(ctx context.Context, input DeleteCommentInput) (any, error) {
	return nil, ErrNotImplemented
}
