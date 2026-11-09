package models

import (
	"time"

	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

type Sprint struct {
	BoardKey  string             `json:"board_key"`
	Number    int                `json:"number"`
	Name      *string            `json:"name,omitempty"`
	Goal      *string            `json:"goal,omitempty"`
	Status    enums.SprintStatus `json:"status"`
	StartDate *time.Time         `json:"start_date,omitempty"`
	EndDate   *time.Time         `json:"end_date,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}
