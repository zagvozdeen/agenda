package models

import (
	"time"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

type Board struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	IsPrivate   bool       `json:"is_private"`
	TaskSeq     int        `json:"task_seq"`
	SprintSeq   int        `json:"sprint_seq"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type BoardMember struct {
	BoardKey string          `json:"board_key"`
	UserID   uuid.UUID       `json:"user_id"`
	Role     enums.BoardRole `json:"role"`
	AddedAt  time.Time       `json:"added_at"`
}
