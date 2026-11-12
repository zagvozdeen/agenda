package models

import (
	"time"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

type CodeLink struct {
	ID        uuid.UUID          `json:"id"`
	TaskID    string             `json:"task_id"`
	Type      enums.CodeLinkType `json:"type"`
	URL       string             `json:"url"`
	BaseSHA   *string            `json:"base_sha,omitempty"`
	HeadSHA   *string            `json:"head_sha,omitempty"`
	Additions *int               `json:"additions"`
	Deletions *int               `json:"deletions"`
	CreatedBy uuid.UUID          `json:"created_by"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type CodeSummary struct {
	Links     int    `json:"links"`
	WithStats int    `json:"with_stats"`
	Additions *int64 `json:"additions"`
	Deletions *int64 `json:"deletions"`
}

type TaskSummary struct {
	Task
	CodeChanges CodeSummary `json:"code_changes"`
}
