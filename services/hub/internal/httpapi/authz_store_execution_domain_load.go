package httpapi

import (
	"database/sql"
	"encoding/json"
	"strings"
)

type executionDomainSnapshot struct {
	Conversations            []Conversation
	ConversationMessages     []ConversationMessage
	ConversationSnapshots    []ConversationSnapshot
	Executions               []Execution
	ExecutionEvents          []ExecutionEvent
	ExecutionControlCommands []ExecutionControlCommand
	ExecutionLeases          []ExecutionLease
	Workers                  []WorkerRegistration
}

func (s *authzStore) loadExecutionDomainSnapshot() (executionDomainSnapshot, error) {
	snapshot := executionDomainSnapshot{
		Conversations:            []Conversation{},
		ConversationMessages:     []ConversationMessage{},
		ConversationSnapshots:    []ConversationSnapshot{},
		Executions:               []Execution{},
		ExecutionEvents:          []ExecutionEvent{},
		ExecutionControlCommands: []ExecutionControlCommand{},
		ExecutionLeases:          []ExecutionLease{},
		Workers:                  []WorkerRegistration{},
	}

	conversationRows, err := s.db.Query(
		`SELECT id, workspace_id, project_id, name, queue_state, default_mode, model_id, base_revision, active_execution_id, created_at, updated_at
		 FROM conversations
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for conversationRows.Next() {
		item := Conversation{}
		var (
			queueStateRaw      string
			defaultModeRaw     string
			activeExecutionRaw sql.NullString
		)
		if err := conversationRows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.ProjectID,
			&item.Name,
			&queueStateRaw,
			&defaultModeRaw,
			&item.ModelID,
			&item.BaseRevision,
			&activeExecutionRaw,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			_ = conversationRows.Close()
			return snapshot, err
		}
		item.QueueState = QueueState(strings.TrimSpace(queueStateRaw))
		item.DefaultMode = ConversationMode(strings.TrimSpace(defaultModeRaw))
		if activeExecutionRaw.Valid && strings.TrimSpace(activeExecutionRaw.String) != "" {
			value := strings.TrimSpace(activeExecutionRaw.String)
			item.ActiveExecutionID = &value
		}
		snapshot.Conversations = append(snapshot.Conversations, item)
	}
	if err := conversationRows.Err(); err != nil {
		_ = conversationRows.Close()
		return snapshot, err
	}
	_ = conversationRows.Close()

	messageRows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, queue_index, can_rollback, created_at
		 FROM conversation_messages
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for messageRows.Next() {
		item := ConversationMessage{}
		var (
			roleRaw        string
			queueIndexRaw  sql.NullInt64
			canRollbackRaw sql.NullInt64
		)
		if err := messageRows.Scan(
			&item.ID,
			&item.ConversationID,
			&roleRaw,
			&item.Content,
			&queueIndexRaw,
			&canRollbackRaw,
			&item.CreatedAt,
		); err != nil {
			_ = messageRows.Close()
			return snapshot, err
		}
		item.Role = MessageRole(strings.TrimSpace(roleRaw))
		if queueIndexRaw.Valid {
			value := int(queueIndexRaw.Int64)
			item.QueueIndex = &value
		}
		if canRollbackRaw.Valid {
			value := canRollbackRaw.Int64 != 0
			item.CanRollback = &value
		}
		snapshot.ConversationMessages = append(snapshot.ConversationMessages, item)
	}
	if err := messageRows.Err(); err != nil {
		_ = messageRows.Close()
		return snapshot, err
	}
	_ = messageRows.Close()

	snapshotRows, err := s.db.Query(
		`SELECT id, conversation_id, rollback_point_message_id, queue_state, worktree_ref, inspector_state_json, messages_json, execution_ids_json, created_at
		 FROM conversation_snapshots
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for snapshotRows.Next() {
		item := ConversationSnapshot{}
		var (
			queueStateRaw    string
			worktreeRefRaw   sql.NullString
			inspectorJSON    string
			messagesJSON     string
			executionIDsJSON string
		)
		if err := snapshotRows.Scan(
			&item.ID,
			&item.ConversationID,
			&item.RollbackPointMessageID,
			&queueStateRaw,
			&worktreeRefRaw,
			&inspectorJSON,
			&messagesJSON,
			&executionIDsJSON,
			&item.CreatedAt,
		); err != nil {
			_ = snapshotRows.Close()
			return snapshot, err
		}
		item.QueueState = QueueState(strings.TrimSpace(queueStateRaw))
		if worktreeRefRaw.Valid && strings.TrimSpace(worktreeRefRaw.String) != "" {
			value := strings.TrimSpace(worktreeRefRaw.String)
			item.WorktreeRef = &value
		}
		if strings.TrimSpace(inspectorJSON) != "" {
			if err := json.Unmarshal([]byte(inspectorJSON), &item.InspectorState); err != nil {
				_ = snapshotRows.Close()
				return snapshot, err
			}
		}
		if strings.TrimSpace(messagesJSON) != "" {
			if err := json.Unmarshal([]byte(messagesJSON), &item.Messages); err != nil {
				_ = snapshotRows.Close()
				return snapshot, err
			}
		}
		if strings.TrimSpace(executionIDsJSON) != "" {
			if err := json.Unmarshal([]byte(executionIDsJSON), &item.ExecutionIDs); err != nil {
				_ = snapshotRows.Close()
				return snapshot, err
			}
		}
		snapshot.ConversationSnapshots = append(snapshot.ConversationSnapshots, item)
	}
	if err := snapshotRows.Err(); err != nil {
		_ = snapshotRows.Close()
		return snapshot, err
	}
	_ = snapshotRows.Close()

	executionRows, err := s.db.Query(
		`SELECT id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json, project_revision_snapshot, queue_index, trace_id, created_at, updated_at
		 FROM executions
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for executionRows.Next() {
		item := Execution{}
		var (
			stateRaw          string
			modeRaw           string
			modeSnapshotRaw   string
			modelSnapshotJSON string
		)
		if err := executionRows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.ConversationID,
			&item.MessageID,
			&stateRaw,
			&modeRaw,
			&item.ModelID,
			&modeSnapshotRaw,
			&modelSnapshotJSON,
			&item.ProjectRevisionSnapshot,
			&item.QueueIndex,
			&item.TraceID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			_ = executionRows.Close()
			return snapshot, err
		}
		item.State = ExecutionState(strings.TrimSpace(stateRaw))
		item.Mode = ConversationMode(strings.TrimSpace(modeRaw))
		item.ModeSnapshot = ConversationMode(strings.TrimSpace(modeSnapshotRaw))
		if strings.TrimSpace(modelSnapshotJSON) != "" {
			if err := json.Unmarshal([]byte(modelSnapshotJSON), &item.ModelSnapshot); err != nil {
				_ = executionRows.Close()
				return snapshot, err
			}
		}
		snapshot.Executions = append(snapshot.Executions, item)
	}
	if err := executionRows.Err(); err != nil {
		_ = executionRows.Close()
		return snapshot, err
	}
	_ = executionRows.Close()

	eventRows, err := s.db.Query(
		`SELECT event_id, execution_id, conversation_id, trace_id, sequence, queue_index, type, timestamp, payload_json
		 FROM execution_events
		 ORDER BY conversation_id ASC, sequence ASC, event_id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for eventRows.Next() {
		item := ExecutionEvent{}
		var (
			eventTypeRaw string
			payloadJSON  string
		)
		if err := eventRows.Scan(
			&item.EventID,
			&item.ExecutionID,
			&item.ConversationID,
			&item.TraceID,
			&item.Sequence,
			&item.QueueIndex,
			&eventTypeRaw,
			&item.Timestamp,
			&payloadJSON,
		); err != nil {
			_ = eventRows.Close()
			return snapshot, err
		}
		item.Type = ExecutionEventType(strings.TrimSpace(eventTypeRaw))
		if strings.TrimSpace(payloadJSON) != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				_ = eventRows.Close()
				return snapshot, err
			}
		}
		if item.Payload == nil {
			item.Payload = map[string]any{}
		}
		snapshot.ExecutionEvents = append(snapshot.ExecutionEvents, item)
	}
	if err := eventRows.Err(); err != nil {
		_ = eventRows.Close()
		return snapshot, err
	}
	_ = eventRows.Close()

	controlRows, err := s.db.Query(
		`SELECT id, execution_id, type, payload_json, seq, created_at
		 FROM execution_control_commands
		 ORDER BY execution_id ASC, seq ASC, id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for controlRows.Next() {
		item := ExecutionControlCommand{}
		var (
			typeRaw     string
			payloadJSON string
		)
		if err := controlRows.Scan(
			&item.ID,
			&item.ExecutionID,
			&typeRaw,
			&payloadJSON,
			&item.Seq,
			&item.CreatedAt,
		); err != nil {
			_ = controlRows.Close()
			return snapshot, err
		}
		item.Type = ExecutionControlCommandType(strings.TrimSpace(typeRaw))
		if strings.TrimSpace(payloadJSON) != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				_ = controlRows.Close()
				return snapshot, err
			}
		}
		if item.Payload == nil {
			item.Payload = map[string]any{}
		}
		snapshot.ExecutionControlCommands = append(snapshot.ExecutionControlCommands, item)
	}
	if err := controlRows.Err(); err != nil {
		_ = controlRows.Close()
		return snapshot, err
	}
	_ = controlRows.Close()

	leaseRows, err := s.db.Query(
		`SELECT execution_id, worker_id, lease_version, lease_expires_at, run_attempt
		 FROM execution_leases
		 ORDER BY execution_id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for leaseRows.Next() {
		item := ExecutionLease{}
		if err := leaseRows.Scan(
			&item.ExecutionID,
			&item.WorkerID,
			&item.LeaseVersion,
			&item.LeaseExpiresAt,
			&item.RunAttempt,
		); err != nil {
			_ = leaseRows.Close()
			return snapshot, err
		}
		snapshot.ExecutionLeases = append(snapshot.ExecutionLeases, item)
	}
	if err := leaseRows.Err(); err != nil {
		_ = leaseRows.Close()
		return snapshot, err
	}
	_ = leaseRows.Close()

	workerRows, err := s.db.Query(
		`SELECT worker_id, capabilities_json, status, last_heartbeat
		 FROM workers
		 ORDER BY worker_id ASC`,
	)
	if err != nil {
		return snapshot, err
	}
	for workerRows.Next() {
		item := WorkerRegistration{}
		var capabilitiesJSON string
		if err := workerRows.Scan(
			&item.WorkerID,
			&capabilitiesJSON,
			&item.Status,
			&item.LastHeartbeat,
		); err != nil {
			_ = workerRows.Close()
			return snapshot, err
		}
		if strings.TrimSpace(capabilitiesJSON) != "" {
			if err := json.Unmarshal([]byte(capabilitiesJSON), &item.Capabilities); err != nil {
				_ = workerRows.Close()
				return snapshot, err
			}
		}
		if item.Capabilities == nil {
			item.Capabilities = map[string]any{}
		}
		snapshot.Workers = append(snapshot.Workers, item)
	}
	if err := workerRows.Err(); err != nil {
		_ = workerRows.Close()
		return snapshot, err
	}
	_ = workerRows.Close()

	return snapshot, nil
}
