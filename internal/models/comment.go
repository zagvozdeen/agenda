package models

import (
	"time"
	"uuid"
)

type Comment struct {
	ID        uuid.UUID  `json:"id"`
	TaskID    string     `json:"task_id"`
	AuthorID  uuid.UUID  `json:"author_id"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
