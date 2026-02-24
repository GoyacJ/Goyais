package httpapi

import (
	"encoding/json"
	"strings"
)

func (s *authzStore) upsertWorkerRegistration(input WorkerRegistration) error {
	capabilitiesJSON, err := json.Marshal(input.Capabilities)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO workers(worker_id, capabilities_json, status, last_heartbeat)
		 VALUES(?,?,?,?)
		 ON CONFLICT(worker_id) DO UPDATE SET
		   capabilities_json=excluded.capabilities_json,
		   status=excluded.status,
		   last_heartbeat=excluded.last_heartbeat`,
		strings.TrimSpace(input.WorkerID),
		string(capabilitiesJSON),
		strings.TrimSpace(input.Status),
		strings.TrimSpace(input.LastHeartbeat),
	)
	return err
}

func (s *authzStore) upsertExecutionLease(input ExecutionLease) error {
	_, err := s.db.Exec(
		`INSERT INTO execution_leases(execution_id, worker_id, lease_version, lease_expires_at, run_attempt)
		 VALUES(?,?,?,?,?)
		 ON CONFLICT(execution_id) DO UPDATE SET
		   worker_id=excluded.worker_id,
		   lease_version=excluded.lease_version,
		   lease_expires_at=excluded.lease_expires_at,
		   run_attempt=excluded.run_attempt`,
		strings.TrimSpace(input.ExecutionID),
		strings.TrimSpace(input.WorkerID),
		input.LeaseVersion,
		strings.TrimSpace(input.LeaseExpiresAt),
		input.RunAttempt,
	)
	return err
}
