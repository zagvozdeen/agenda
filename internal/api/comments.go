package api

import (
	"context"
	"strings"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

func (a *API) registerComments() {
	registerTool(a, "add_comment", "Add a task comment. Requires board membership.", false, false, a.AddComment)
	registerTool(a, "update_comment", "Update your own comment. Requires board membership.", false, false, a.UpdateComment)
	registerTool(a, "delete_comment", "Soft-delete your own comment. Requires board membership.", false, false, a.DeleteComment)
}

func (a *API) AddComment(ctx context.Context, input AddCommentInput) (any, error) {
	if strings.TrimSpace(input.Body) == "" {
		return nil, invalid("comment body must not be empty")
	}
	if _, err := a.writableTask(ctx, input.TaskID); err != nil {
		return nil, err
	}
	comment := &models.Comment{TaskID: input.TaskID, AuthorID: actor(ctx).ID, Body: input.Body}
	if err := a.store.CreateComment(ctx, comment); err != nil {
		return nil, err
	}
	if err := a.event(ctx, comment.TaskID, enums.EventTypeCommented, "comment", nil, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (a *API) UpdateComment(ctx context.Context, input UpdateCommentInput) (any, error) {
	if strings.TrimSpace(input.Body) == "" {
		return nil, invalid("comment body must not be empty")
	}
	comment, err := a.store.GetComment(ctx, input.CommentID)
	if err != nil {
		return nil, err
	}
	if _, err = a.writableTask(ctx, comment.TaskID); err != nil {
		return nil, err
	}
	comment, err = a.store.GetComment(ctx, input.CommentID)
	if err != nil {
		return nil, err
	}
	if comment.AuthorID != actor(ctx).ID {
		return nil, forbidden("only the comment author may edit it")
	}
	if comment.Body == input.Body {
		return comment, nil
	}
	before := *comment
	comment.Body = input.Body
	if err = a.store.UpdateComment(ctx, comment); err != nil {
		return nil, err
	}
	if err = a.event(ctx, comment.TaskID, enums.EventTypeUpdated, "comment", before, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (a *API) DeleteComment(ctx context.Context, input DeleteCommentInput) (any, error) {
	comment, err := a.store.GetComment(ctx, input.CommentID)
	if err != nil {
		return nil, err
	}
	if _, err = a.writableTask(ctx, comment.TaskID); err != nil {
		return nil, err
	}
	comment, err = a.store.GetComment(ctx, input.CommentID)
	if err != nil {
		return nil, err
	}
	if comment.AuthorID != actor(ctx).ID {
		return nil, forbidden("only the comment author may delete it")
	}
	before := *comment
	if err = a.store.DeleteComment(ctx, comment); err != nil {
		return nil, err
	}
	if err = a.event(ctx, comment.TaskID, enums.EventTypeDeleted, "comment", before, nil); err != nil {
		return nil, err
	}
	return comment, nil
}
