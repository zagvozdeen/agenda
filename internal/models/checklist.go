package models

import (
	"time"
	"uuid"
)

type ChecklistItem struct {
	ID        uuid.UUID `json:"id"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
