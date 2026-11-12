package models

import "gitverse.ru/zagvozdeen/agenda/internal/models/enums"

type Membership struct {
	BoardKey string          `json:"board_key"`
	Name     string          `json:"name"`
	Role     enums.BoardRole `json:"role"`
}

type Member struct {
	BoardMember
	Username string `json:"username"`
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

type BoardView struct {
	Board
	Members      []Member                 `json:"members"`
	ActiveSprint *Sprint                  `json:"active_sprint"`
	TaskCounts   map[enums.TaskStatus]int `json:"task_counts"`
}

type TaskView struct {
	Task
	CodeLinks   []CodeLink  `json:"code_links"`
	CodeChanges CodeSummary `json:"code_changes"`
	Subtasks    []Task      `json:"subtasks"`
	Links       []TaskLink  `json:"links"`
	Comments    []Comment   `json:"comments"`
	Events      []TaskEvent `json:"events"`
}
