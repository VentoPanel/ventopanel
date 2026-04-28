package audit

import (
	"time"

	domain "github.com/your-org/ventopanel/internal/domain/audit"
)

type Service struct {
	reader domain.StatusEventReader
}

type Filter struct {
	ResourceType string
	ResourceID   string
	FromStatus   string
	ToStatus     string
	Since        *time.Time
	BeforeCreatedAt *time.Time
	BeforeID     string
	Limit        int
	IncludeTotal bool
}

type Result struct {
	Items      []domain.StatusEvent
	TotalCount *int64
}

func NewService(reader domain.StatusEventReader) *Service {
	return &Service{reader: reader}
}

func (s *Service) List(filter Filter) (Result, error) {
	base := domain.StatusEventFilter{
		ResourceType: filter.ResourceType,
		ResourceID:   filter.ResourceID,
		FromStatus:   filter.FromStatus,
		ToStatus:     filter.ToStatus,
		Since:        filter.Since,
		BeforeCreatedAt: filter.BeforeCreatedAt,
		BeforeID:     filter.BeforeID,
		Limit:        filter.Limit,
	}

	items, err := s.reader.ListStatusEvents(base)
	if err != nil {
		return Result{}, err
	}

	result := Result{Items: items}
	if filter.IncludeTotal {
		// Count ignores cursor to represent full filtered dataset size.
		countFilter := base
		countFilter.BeforeCreatedAt = nil
		countFilter.BeforeID = ""
		total, err := s.reader.CountStatusEvents(countFilter)
		if err != nil {
			return Result{}, err
		}
		result.TotalCount = &total
	}

	return result, nil
}
