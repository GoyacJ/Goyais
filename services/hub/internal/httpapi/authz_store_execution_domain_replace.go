package httpapi

import "encoding/json"

func (s *authzStore) replaceExecutionDomainSnapshot(snapshot executionDomainSnapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	cleanupStatements := []string{
		`DELETE FROM execution_control_commands`,
		`DELETE FROM execution_leases`,
		`DELETE FROM workers`,
		`DELETE FROM execution_events`,
		`DELETE FROM executions`,
		`DELETE FROM conversation_snapshots`,
		`DELETE FROM conversation_messages`,
		`DELETE FROM conversations`,
	}
	for _, statement := range cleanupStatements {
		if _, err = tx.Exec(statement); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Conversations {
		if _, err = tx.Exec(
			`INSERT INTO conversations(id, workspace_id, project_id, name, queue_state, default_mode, model_id, base_revision, active_execution_id, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.WorkspaceID,
			item.ProjectID,
			item.Name,
			string(item.QueueState),
			string(item.DefaultMode),
			item.ModelID,
			item.BaseRevision,
			nullWhenEmpty(derefString(item.ActiveExecutionID)),
			item.CreatedAt,
			item.UpdatedAt,
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.ConversationMessages {
		if _, err = tx.Exec(
			`INSERT INTO conversation_messages(id, conversation_id, role, content, queue_index, can_rollback, created_at)
			 VALUES(?,?,?,?,?,?,?)`,
			item.ID,
			item.ConversationID,
			string(item.Role),
			item.Content,
			nullableInt(item.QueueIndex),
			nullableBool(item.CanRollback),
			item.CreatedAt,
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.ConversationSnapshots {
		inspectorJSON, marshalErr := json.Marshal(item.InspectorState)
		if marshalErr != nil {
			return marshalErr
		}
		messagesJSON, marshalErr := json.Marshal(item.Messages)
		if marshalErr != nil {
			return marshalErr
		}
		executionIDsJSON, marshalErr := json.Marshal(item.ExecutionIDs)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(
			`INSERT INTO conversation_snapshots(id, conversation_id, rollback_point_message_id, queue_state, worktree_ref, inspector_state_json, messages_json, execution_ids_json, created_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.ConversationID,
			item.RollbackPointMessageID,
			string(item.QueueState),
			nullWhenEmpty(derefString(item.WorktreeRef)),
			string(inspectorJSON),
			string(messagesJSON),
			string(executionIDsJSON),
			item.CreatedAt,
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Executions {
		modelSnapshotJSON, marshalErr := json.Marshal(item.ModelSnapshot)
		if marshalErr != nil {
			return marshalErr
		}
		var agentConfigSnapshotJSON any
		if item.AgentConfigSnapshot != nil {
			encoded, encodeErr := json.Marshal(item.AgentConfigSnapshot)
			if encodeErr != nil {
				return encodeErr
			}
			agentConfigSnapshotJSON = string(encoded)
		}
		if _, err = tx.Exec(
			`INSERT INTO executions(id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot, queue_index, trace_id, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			item.ID,
			item.WorkspaceID,
			item.ConversationID,
			item.MessageID,
			string(item.State),
			string(item.Mode),
			item.ModelID,
			string(item.ModeSnapshot),
			string(modelSnapshotJSON),
			agentConfigSnapshotJSON,
			item.TokensIn,
			item.TokensOut,
			item.ProjectRevisionSnapshot,
			item.QueueIndex,
			item.TraceID,
			item.CreatedAt,
			item.UpdatedAt,
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.ExecutionEvents {
		payloadJSON, marshalErr := json.Marshal(item.Payload)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(
			`INSERT INTO execution_events(event_id, execution_id, conversation_id, trace_id, sequence, queue_index, type, timestamp, payload_json)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			item.EventID,
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

	for _, item := range snapshot.ExecutionControlCommands {
		payloadJSON, marshalErr := json.Marshal(item.Payload)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(
			`INSERT INTO execution_control_commands(id, execution_id, type, payload_json, seq, created_at)
			 VALUES(?,?,?,?,?,?)`,
			item.ID,
			item.ExecutionID,
			string(item.Type),
			string(payloadJSON),
			item.Seq,
			item.CreatedAt,
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.ExecutionLeases {
		if _, err = tx.Exec(
			`INSERT INTO execution_leases(execution_id, worker_id, lease_version, lease_expires_at, run_attempt)
			 VALUES(?,?,?,?,?)`,
			item.ExecutionID,
			item.WorkerID,
			item.LeaseVersion,
			item.LeaseExpiresAt,
			item.RunAttempt,
		); err != nil {
			return err
		}
	}

	for _, item := range snapshot.Workers {
		capabilitiesJSON, marshalErr := json.Marshal(item.Capabilities)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(
			`INSERT INTO workers(worker_id, capabilities_json, status, last_heartbeat)
			 VALUES(?,?,?,?)`,
			item.WorkerID,
			string(capabilitiesJSON),
			item.Status,
			item.LastHeartbeat,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	if *value {
		return 1
	}
	return 0
}
