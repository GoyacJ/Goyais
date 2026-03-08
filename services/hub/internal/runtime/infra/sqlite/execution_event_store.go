package sqlite

import (
	"database/sql"
	"encoding/json"
	"strings"

	runtimedomain "goyais/services/hub/internal/runtime/domain"
)

type queryExecutor interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

type ExecutionEventStore struct {
	executor queryExecutor
}

func NewExecutionEventStore(db *sql.DB) *ExecutionEventStore {
	return &ExecutionEventStore{executor: db}
}

func NewExecutionEventStoreWithTx(tx *sql.Tx) *ExecutionEventStore {
	return &ExecutionEventStore{executor: tx}
}

func (s *ExecutionEventStore) LoadAll() ([]runtimedomain.Event, error) {
	if s == nil || s.executor == nil {
		return []runtimedomain.Event{}, nil
	}
	rows, err := s.executor.Query(
		`SELECT event_id, execution_id, conversation_id, trace_id, sequence, queue_index, type, timestamp, payload_json
		 FROM execution_events
		 ORDER BY conversation_id ASC, sequence ASC, event_id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []runtimedomain.Event{}
	for rows.Next() {
		item := runtimedomain.Event{}
		var (
			eventTypeRaw string
			payloadJSON  string
		)
		if err := rows.Scan(
			&item.ID,
			&item.ExecutionID,
			&item.ConversationID,
			&item.TraceID,
			&item.Sequence,
			&item.QueueIndex,
			&eventTypeRaw,
			&item.Timestamp,
			&payloadJSON,
		); err != nil {
			return nil, err
		}
		item.Type = runtimedomain.EventType(strings.TrimSpace(eventTypeRaw))
		if strings.TrimSpace(payloadJSON) != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return nil, err
			}
		}
		if item.Payload == nil {
			item.Payload = map[string]any{}
		}
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *ExecutionEventStore) ReplaceAll(events []runtimedomain.Event) error {
	if s == nil || s.executor == nil {
		return nil
	}
	if _, err := s.executor.Exec(`DELETE FROM execution_events`); err != nil {
		return err
	}
	for _, item := range events {
		payload := item.Payload
		if payload == nil {
			payload = map[string]any{}
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := s.executor.Exec(
			`INSERT INTO execution_events(event_id, execution_id, conversation_id, trace_id, sequence, queue_index, type, timestamp, payload_json)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.ExecutionID,
			item.ConversationID,
			item.TraceID,
			item.Sequence,
			item.QueueIndex,
			string(item.Type),
			item.Timestamp,
			string(payloadJSON),
		); err != nil {
			return err
		}
	}
	return nil
}
