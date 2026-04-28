package audit

import "time"

type StatusEvent struct {
	ID           string
	ResourceType string
	ResourceID   string
	FromStatus   string
	ToStatus     string
	Reason       string
	TaskID       string
	CreatedAt    time.Time
}

type StatusEventWriter interface {
	WriteStatusEvent(event StatusEvent) error
}

type StatusEventFilter struct {
	ResourceType string
	ResourceID   string
	FromStatus   string
	ToStatus     string
	Since        *time.Time
	BeforeCreatedAt *time.Time
	BeforeID     string
	Limit        int
}

type StatusEventReader interface {
	ListStatusEvents(filter StatusEventFilter) ([]StatusEvent, error)
	CountStatusEvents(filter StatusEventFilter) (int64, error)
}
