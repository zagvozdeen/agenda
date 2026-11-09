package enums

type TaskType string

const (
	TaskTypeFeature TaskType = "feature"
	TaskTypeBug     TaskType = "bug"
	TaskTypeChore   TaskType = "chore"
)
