package httpapi

import (
	"database/sql"
	"encoding/json"
	"strings"
)

type executionDomainSnapshot struct {
	Conversations         []Conversation
	ConversationMessages  []ConversationMessage
	ConversationSnapshots []ConversationSnapshot
	Executions            []Execution
	ExecutionEvents       []ExecutionEvent
}

func (s *authzStore) loadExecutionDomainSnapshot() (executionDomainSnapshot, error) {
	snapshot := executionDomainSnapshot{
		Conversations:         []Conversation{},
		ConversationMessages:  []ConversationMessage{},
		ConversationSnapshots: []ConversationSnapshot{},
		Executions:            []Execution{},
		ExecutionEvents:       []ExecutionEvent{},
	}

	conversationRows, err := s.db.Query(
		`SELECT id, workspace_id, project_id, name, queue_state, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, base_revision, active_execution_id, created_at, updated_at
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
			ruleIDsJSON        string
			skillIDsJSON       string
			mcpIDsJSON         string
			activeExecutionRaw sql.NullString
		)
		if err := conversationRows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.ProjectID,
			&item.Name,
			&queueStateRaw,
			&defaultModeRaw,
			&item.ModelConfigID,
			&ruleIDsJSON,
			&skillIDsJSON,
			&mcpIDsJSON,
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
		if strings.TrimSpace(ruleIDsJSON) != "" {
			if err := json.Unmarshal([]byte(ruleIDsJSON), &item.RuleIDs); err != nil {
				_ = conversationRows.Close()
				return snapshot, err
			}
		}
		if item.RuleIDs == nil {
			item.RuleIDs = []string{}
		}
		if strings.TrimSpace(skillIDsJSON) != "" {
			if err := json.Unmarshal([]byte(skillIDsJSON), &item.SkillIDs); err != nil {
				_ = conversationRows.Close()
				return snapshot, err
			}
		}
		if item.SkillIDs == nil {
			item.SkillIDs = []string{}
		}
		if strings.TrimSpace(mcpIDsJSON) != "" {
			if err := json.Unmarshal([]byte(mcpIDsJSON), &item.MCPIDs); err != nil {
				_ = conversationRows.Close()
				return snapshot, err
			}
		}
		if item.MCPIDs == nil {
			item.MCPIDs = []string{}
		}
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
		`SELECT id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json, resource_profile_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot, queue_index, trace_id, created_at, updated_at
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
			resourceJSON      sql.NullString
			agentConfigJSON   sql.NullString
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
			&resourceJSON,
			&agentConfigJSON,
			&item.TokensIn,
			&item.TokensOut,
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
		if resourceJSON.Valid && strings.TrimSpace(resourceJSON.String) != "" {
			resourceSnapshot := ExecutionResourceProfile{}
			if err := json.Unmarshal([]byte(resourceJSON.String), &resourceSnapshot); err != nil {
				_ = executionRows.Close()
				return snapshot, err
			}
			item.ResourceProfileSnapshot = &resourceSnapshot
		}
		if agentConfigJSON.Valid && strings.TrimSpace(agentConfigJSON.String) != "" {
			configSnapshot := ExecutionAgentConfigSnapshot{}
			if err := json.Unmarshal([]byte(agentConfigJSON.String), &configSnapshot); err != nil {
				_ = executionRows.Close()
				return snapshot, err
			}
			item.AgentConfigSnapshot = &configSnapshot
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

	return snapshot, nil
}
