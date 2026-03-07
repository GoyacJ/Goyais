package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"goyais/services/hub/internal/domain"
)

type RunEventRepository struct {
	db *sql.DB
}

func NewRunEventRepository(db *sql.DB) RunEventRepository {
	return RunEventRepository{db: db}
}

func (r RunEventRepository) Append(ctx context.Context, event domain.RunEvent) error {
	if r.db == nil {
		return nil
	}
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO domain_run_events(event_id, run_id, session_id, sequence, type, payload_json, occurred_at)
		 VALUES(?,?,?,?,?,?,?)`,
		event.EventID,
		string(event.RunID),
		string(event.SessionID),
		event.Sequence,
		event.Type,
		string(payloadJSON),
		event.OccurredAt,
	)
	return err
}

func (r RunEventRepository) ListBySessionSince(ctx context.Context, sessionID domain.SessionID, afterSequence int64, limit int) ([]domain.RunEvent, error) {
	if r.db == nil {
		return []domain.RunEvent{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT event_id, run_id, session_id, sequence, type, payload_json, occurred_at
		   FROM domain_run_events
		  WHERE session_id = ? AND sequence > ?
		  ORDER BY sequence ASC, event_id ASC
		  LIMIT ?`,
		string(sessionID),
		afterSequence,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.RunEvent{}
	for rows.Next() {
		item := domain.RunEvent{}
		var payloadJSON string
		if err := rows.Scan(
			&item.EventID,
			&item.RunID,
			&item.SessionID,
			&item.Sequence,
			&item.Type,
			&payloadJSON,
			&item.OccurredAt,
		); err != nil {
			return nil, err
		}
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
