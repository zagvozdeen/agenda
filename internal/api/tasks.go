package api

import "context"

func (a *API) registerTasks() {
	registerTool(a, "create_task", "Create a task or subtask with an automatically assigned ID. Requires board membership.", false, false, a.CreateTask)
	registerTool(a, "get_task", "Get a task with subtasks, incoming and outgoing links, comments and task events.", true, true, a.GetTask)
	registerTool(a, "update_task", "Update mutable task fields. Board key and task number are immutable. Requires board membership.", false, false, a.UpdateTask)
	registerTool(a, "delete_task", "Soft-delete a task. Delete its non-deleted children first. Requires board membership.", false, false, a.DeleteTask)
	registerTool(a, "search_tasks", "Search current task fields with filters, full-text search and sorting. Historical assignments and sprint membership are not searched.", true, true, a.SearchTasks)
	registerTool(a, "link_tasks", "Create a directed relationship between tasks. Requires board membership.", false, false, a.LinkTasks)
	registerTool(a, "unlink_tasks", "Remove a relationship between tasks. Requires board membership.", false, false, a.UnlinkTasks)
}

func (a *API) CreateTask(ctx context.Context, input CreateTaskInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) GetTask(ctx context.Context, input TaskInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) UpdateTask(ctx context.Context, input UpdateTaskInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) DeleteTask(ctx context.Context, input TaskInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) SearchTasks(ctx context.Context, input SearchTasksInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) LinkTasks(ctx context.Context, input TaskLinkInput) (any, error) {
	return nil, ErrNotImplemented
}

func (a *API) UnlinkTasks(ctx context.Context, input TaskLinkInput) (any, error) {
	return nil, ErrNotImplemented
}
