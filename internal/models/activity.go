package models

import (
	"encoding/json/jsontext"
	"time"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

type TaskEvent struct {
	ID        uuid.UUID       `json:"id"`
	TaskID    string          `json:"task_id"`
	ActorID   uuid.UUID       `json:"actor_id"`
	EventType enums.EventType `json:"event_type"`
	Field     *string         `json:"field,omitempty"`
	OldValue  jsontext.Value  `json:"old_value,omitempty"`
	NewValue  jsontext.Value  `json:"new_value,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
