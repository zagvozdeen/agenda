package enums

type SprintStatus string

const (
	SprintStatusPlanned SprintStatus = "planned"
	SprintStatusActive  SprintStatus = "active"
	SprintStatusClosed  SprintStatus = "closed"
)
