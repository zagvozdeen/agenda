package models

import (
	"time"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

type Task struct {
	ID             string             `json:"id"`
	BoardKey       string             `json:"board_key"`
	Number         int                `json:"number"`
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	Type           enums.TaskType     `json:"type"`
	Status         enums.TaskStatus   `json:"status"`
	Priority       enums.TaskPriority `json:"priority"`
	AssigneeID     *uuid.UUID         `json:"assignee_id,omitempty"`
	CreatedBy      uuid.UUID          `json:"created_by"`
	SprintNumber   *int               `json:"sprint_number,omitempty"`
	ParentID       *string            `json:"parent_id,omitempty"`
	EstimateHours  *int               `json:"estimate_hours,omitempty"`
	IdempotencyKey *uuid.UUID         `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
	DeletedAt      *time.Time         `json:"deleted_at,omitempty"`
}

type TaskLink struct {
	FromTaskID string         `json:"from_task_id"`
	ToTaskID   string         `json:"to_task_id"`
	Type       enums.LinkType `json:"type"`
	CreatedBy  uuid.UUID      `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
}
