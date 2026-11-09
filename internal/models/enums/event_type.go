package enums

type EventType string

const (
	EventTypeCreated   EventType = "created"
	EventTypeUpdated   EventType = "updated"
	EventTypeDeleted   EventType = "deleted"
	EventTypeCommented EventType = "commented"
	EventTypeLinked    EventType = "linked"
	EventTypeUnlinked  EventType = "unlinked"
)
