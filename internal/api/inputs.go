package api

import (
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

type EmptyInput struct{}

type FindUsersInput struct {
	Query  string `json:"query" jsonschema:"Search by current display name or username."`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum number of results; default 50, maximum 100."`
	Offset int    `json:"offset,omitempty" jsonschema:"Number of results to skip."`
}

type ListBoardsInput struct {
	MemberID *uuid.UUID `json:"member_id,omitempty" jsonschema:"Filter by current board membership."`
	Limit    int        `json:"limit,omitempty" jsonschema:"Maximum number of results; default 50, maximum 100."`
	Offset   int        `json:"offset,omitempty" jsonschema:"Number of results to skip."`
}

type BoardInput struct {
	BoardKey string `json:"board_key" jsonschema:"Immutable board key consisting of three uppercase Latin letters, for example ABC."`
}

type CreateBoardInput struct {
	BoardKey    string  `json:"board_key" jsonschema:"Immutable board key consisting of three uppercase Latin letters."`
	Name        string  `json:"name" jsonschema:"Board name."`
	Description *string `json:"description,omitempty" jsonschema:"Board description."`
}

type UpdateBoardInput struct {
	BoardKey    string           `json:"board_key" jsonschema:"Board key."`
	Name        *string          `json:"name,omitempty" jsonschema:"New board name."`
	Description Optional[string] `json:"description,omitzero" jsonschema:"New board description."`
}

type AddMemberInput struct {
	BoardKey string          `json:"board_key" jsonschema:"Board key."`
	UserID   uuid.UUID       `json:"user_id" jsonschema:"User to add or change the role of."`
	Role     enums.BoardRole `json:"role" jsonschema:"Board role: owner or member. The last owner cannot be demoted."`
}

type RemoveMemberInput struct {
	BoardKey string    `json:"board_key" jsonschema:"Board key."`
	UserID   uuid.UUID `json:"user_id" jsonschema:"User to remove. The last owner cannot be removed."`
}

type ListSprintsInput struct {
	BoardKey string              `json:"board_key" jsonschema:"Board key."`
	Status   *enums.SprintStatus `json:"status,omitempty" jsonschema:"Filter by sprint status."`
	Limit    int                 `json:"limit,omitempty" jsonschema:"Maximum number of results; default 50, maximum 100."`
	Offset   int                 `json:"offset,omitempty" jsonschema:"Number of results to skip."`
}

type CreateSprintInput struct {
	BoardKey  string  `json:"board_key" jsonschema:"Board key."`
	Name      *string `json:"name,omitempty" jsonschema:"Sprint name."`
	Goal      *string `json:"goal,omitempty" jsonschema:"Sprint goal."`
	StartDate *string `json:"start_date,omitempty" jsonschema:"Planned start date in YYYY-MM-DD format."`
	EndDate   *string `json:"end_date,omitempty" jsonschema:"Planned end date in YYYY-MM-DD format."`
}

type UpdateSprintInput struct {
	BoardKey  string           `json:"board_key" jsonschema:"Board key."`
	Number    int              `json:"number" jsonschema:"Sprint number within the board."`
	Name      Optional[string] `json:"name,omitzero" jsonschema:"New sprint name."`
	Goal      Optional[string] `json:"goal,omitzero" jsonschema:"New sprint goal."`
	StartDate Optional[string] `json:"start_date,omitzero" jsonschema:"Start date in YYYY-MM-DD format."`
	EndDate   Optional[string] `json:"end_date,omitzero" jsonschema:"End date in YYYY-MM-DD format."`
}

type SprintInput struct {
	BoardKey string `json:"board_key" jsonschema:"Board key."`
	Number   int    `json:"number" jsonschema:"Sprint number within the board."`
}

type CloseSprintInput struct {
	BoardKey           string `json:"board_key" jsonschema:"Board key."`
	Number             int    `json:"number" jsonschema:"Sprint to close."`
	TargetSprintNumber *int   `json:"target_sprint_number,omitempty" jsonschema:"Planned sprint for unfinished tasks; omit to move them to backlog. Done and cancelled tasks stay in the closed sprint."`
}

type CreateTaskInput struct {
	BoardKey       string             `json:"board_key" jsonschema:"Board key."`
	Title          string             `json:"title" jsonschema:"Task title."`
	Description    string             `json:"description,omitempty" jsonschema:"Task description in Markdown."`
	Type           enums.TaskType     `json:"type,omitempty" jsonschema:"Task type; default feature."`
	Priority       enums.TaskPriority `json:"priority,omitempty" jsonschema:"Task priority; default medium."`
	AssigneeID     *uuid.UUID         `json:"assignee_id,omitempty" jsonschema:"Current assignee."`
	SprintNumber   *int               `json:"sprint_number,omitempty" jsonschema:"Sprint on this board; omit for backlog."`
	ParentID       *string            `json:"parent_id,omitempty" jsonschema:"Parent task on the same board."`
	EstimateHours  *int               `json:"estimate_hours,omitempty" jsonschema:"Estimated effort in whole hours, at least zero."`
	IdempotencyKey *uuid.UUID         `json:"idempotency_key,omitempty" jsonschema:"Reuse the same UUID when retrying task creation."`
}

type TaskInput struct {
	TaskID string `json:"task_id" jsonschema:"Immutable task ID, for example ABC-12."`
}

type UpdateTaskInput struct {
	TaskID        string              `json:"task_id" jsonschema:"Immutable task ID."`
	Title         *string             `json:"title,omitempty" jsonschema:"New title."`
	Description   *string             `json:"description,omitempty" jsonschema:"New Markdown description."`
	Type          *enums.TaskType     `json:"type,omitempty" jsonschema:"New task type."`
	Status        *enums.TaskStatus   `json:"status,omitempty" jsonschema:"New task status."`
	Priority      *enums.TaskPriority `json:"priority,omitempty" jsonschema:"New task priority."`
	AssigneeID    Optional[uuid.UUID] `json:"assignee_id,omitzero" jsonschema:"New assignee; null clears the assignee."`
	SprintNumber  Optional[int]       `json:"sprint_number,omitzero" jsonschema:"New sprint number on the same board; null moves the task to backlog."`
	ParentID      Optional[string]    `json:"parent_id,omitzero" jsonschema:"New parent task on the same board; null clears the parent."`
	EstimateHours Optional[int]       `json:"estimate_hours,omitzero" jsonschema:"New estimated effort in whole hours; null clears the estimate."`
}

type SearchTasksInput struct {
	BoardKey     *string             `json:"board_key,omitempty" jsonschema:"Filter by board."`
	Query        *string             `json:"query,omitempty" jsonschema:"Full-text search in current title and description."`
	AssigneeID   *uuid.UUID          `json:"assignee_id,omitempty" jsonschema:"Filter by current assignee only; historical assignments are not searched."`
	CreatedBy    *uuid.UUID          `json:"created_by,omitempty" jsonschema:"Filter by task author."`
	SprintNumber *int                `json:"sprint_number,omitempty" jsonschema:"Filter by current sprint only; requires board_key. Historical sprint membership is not searched."`
	ParentID     *string             `json:"parent_id,omitempty" jsonschema:"Filter by parent task."`
	Type         *enums.TaskType     `json:"type,omitempty" jsonschema:"Filter by task type."`
	Status       *enums.TaskStatus   `json:"status,omitempty" jsonschema:"Filter by current task status."`
	Priority     *enums.TaskPriority `json:"priority,omitempty" jsonschema:"Filter by priority."`
	Backlog      bool                `json:"backlog,omitempty" jsonschema:"Only tasks without a sprint."`
	Unassigned   bool                `json:"unassigned,omitempty" jsonschema:"Only tasks without an assignee."`
	Sort         string              `json:"sort,omitempty" jsonschema:"Sort field: created_at, updated_at, priority, number or completed_at. Default updated_at."`
	Direction    string              `json:"direction,omitempty" jsonschema:"Sort direction: asc or desc. Default desc."`
	Limit        int                 `json:"limit,omitempty" jsonschema:"Maximum number of results; default 50, maximum 100."`
	Offset       int                 `json:"offset,omitempty" jsonschema:"Number of results to skip."`
}

type TaskLinkInput struct {
	FromTaskID string         `json:"from_task_id" jsonschema:"Source task ID."`
	ToTaskID   string         `json:"to_task_id" jsonschema:"Target task ID; must differ from the source."`
	Type       enums.LinkType `json:"type" jsonschema:"Directed relationship: source blocks, relates to or duplicates target."`
}

type AddCommentInput struct {
	TaskID string `json:"task_id" jsonschema:"Task to comment on."`
	Body   string `json:"body" jsonschema:"Comment body in Markdown."`
}

type UpdateCommentInput struct {
	CommentID uuid.UUID `json:"comment_id" jsonschema:"ID of your own comment."`
	Body      string    `json:"body" jsonschema:"New Markdown body."`
}

type DeleteCommentInput struct {
	CommentID uuid.UUID `json:"comment_id" jsonschema:"ID of your own comment to soft-delete."`
}

type ListActivityInput struct {
	BoardKey *string    `json:"board_key,omitempty" jsonschema:"Filter task events by board."`
	TaskID   *string    `json:"task_id,omitempty" jsonschema:"Filter events by task."`
	ActorID  *uuid.UUID `json:"actor_id,omitempty" jsonschema:"Filter by the user who performed the change."`
	Limit    int        `json:"limit,omitempty" jsonschema:"Maximum number of results; default 50, maximum 100."`
	Offset   int        `json:"offset,omitempty" jsonschema:"Number of results to skip."`
}
