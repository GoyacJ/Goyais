package httpapi

import (
	"context"
	"database/sql"
	"strings"
)

func (s *authzStore) replaceSessionResourceSnapshots(sessionID string, snapshots []SessionResourceSnapshot) error {
	if s == nil || s.db == nil {
		return nil
	}
	normalizedSessionID := strings.TrimSpace(sessionID)
	return withWriteTx(context.Background(), s.db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(
			`DELETE FROM session_resource_snapshots WHERE session_id=?`,
			normalizedSessionID,
		); err != nil {
			return err
		}
		for _, item := range snapshots {
			payload, err := encodeResourceConfigPayload(item.CapturedConfig)
			if err != nil {
				return err
			}
			var fallbackResourceID any
			if item.FallbackResourceID != nil {
				fallbackResourceID = strings.TrimSpace(*item.FallbackResourceID)
			}
			if _, err := tx.Exec(
				`INSERT INTO session_resource_snapshots(
					session_id, resource_config_id, resource_type, resource_version, is_deprecated, fallback_resource_id, payload_json, snapshot_at
				) VALUES(?,?,?,?,?,?,?,?)`,
				normalizedSessionID,
				strings.TrimSpace(item.ResourceConfigID),
				string(item.ResourceType),
				normalizeResourceConfigVersion(item.ResourceVersion),
				boolToInt(item.IsDeprecated),
				fallbackResourceID,
				payload,
				strings.TrimSpace(item.SnapshotAt),
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *authzStore) listSessionResourceSnapshots(sessionID string) ([]SessionResourceSnapshot, error) {
	if s == nil || s.db == nil {
		return []SessionResourceSnapshot{}, nil
	}
	rows, err := s.db.Query(
		`SELECT session_id, resource_config_id, resource_type, resource_version, is_deprecated, fallback_resource_id, payload_json, snapshot_at
		 FROM session_resource_snapshots
		 WHERE session_id=?
		 ORDER BY snapshot_at ASC, resource_config_id ASC`,
		strings.TrimSpace(sessionID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SessionResourceSnapshot, 0)
	for rows.Next() {
		item := SessionResourceSnapshot{}
		var (
			isDeprecated      int
			fallbackResourceID sql.NullString
			payloadJSON       string
		)
		if err := rows.Scan(
			&item.SessionID,
			&item.ResourceConfigID,
			&item.ResourceType,
			&item.ResourceVersion,
			&isDeprecated,
			&fallbackResourceID,
			&payloadJSON,
			&item.SnapshotAt,
		); err != nil {
			return nil, err
		}
		item.IsDeprecated = isDeprecated != 0
		item.FallbackResourceID = nullStringToPointer(fallbackResourceID)
		capturedConfig, err := decodeResourceConfigPayload(payloadJSON, false)
		if err != nil {
			return nil, err
		}
		item.CapturedConfig = capturedConfig
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *authzStore) deleteSessionResourceSnapshots(sessionID string) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.Exec(
		`DELETE FROM session_resource_snapshots WHERE session_id=?`,
		strings.TrimSpace(sessionID),
	)
	return err
}
