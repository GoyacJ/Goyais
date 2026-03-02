package domain

type EventType string

const EventTypeDiffGenerated EventType = "diff_generated"

type Event struct {
	ID             string
	ConversationID string
	ExecutionID    string
	TraceID        string
	Sequence       int
	QueueIndex     int
	Type           EventType
	Timestamp      string
	Payload        map[string]any
}

type DiffItem struct {
	ID           string
	Path         string
	ChangeType   string
	Summary      string
	AddedLines   *int
	DeletedLines *int
	BeforeBlob   string
	AfterBlob    string
}
