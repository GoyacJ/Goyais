package domain

type RunEvent struct {
	EventID    string
	RunID      RunID
	SessionID  SessionID
	Sequence   int64
	Type       string
	Payload    map[string]any
	OccurredAt string
}
