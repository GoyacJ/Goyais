package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"
	"time"
)

type Service struct {
	repo           Repository
	idempotencyTTL time.Duration
	logger         *log.Logger
}

func NewService(repo Repository, idempotencyTTL time.Duration, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		repo:           repo,
		idempotencyTTL: idempotencyTTL,
		logger:         logger,
	}
}

func (s *Service) Submit(ctx context.Context, reqCtx RequestContext, commandType string, payload json.RawMessage, idempotencyKey string) (Command, error) {
	if err := validateCreateRequest(commandType, payload); err != nil {
		return Command{}, err
	}

	now := time.Now().UTC()
	requestHash := ""
	if idempotencyKey != "" {
		requestHash = hashRequest(commandType, payload)
	} else {
		s.logger.Printf("WARN: missing Idempotency-Key tenant=%s workspace=%s user=%s", reqCtx.TenantID, reqCtx.WorkspaceID, reqCtx.UserID)
	}

	created, err := s.repo.Create(ctx, CreateInput{
		Context:        reqCtx,
		CommandType:    commandType,
		Payload:        payload,
		IdempotencyKey: idempotencyKey,
		RequestHash:    requestHash,
		Now:            now,
		TTL:            s.idempotencyTTL,
	})
	if err != nil {
		return Command{}, err
	}

	if created.Reused {
		return created.Command, nil
	}

	_ = s.repo.AppendCommandEvent(ctx, reqCtx, created.Command.ID, "command.accepted", payload)
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, created.Command.ID, "command.authorize", "allow", "stub_authorizer", payload)

	running, err := s.repo.SetStatus(ctx, reqCtx, created.Command.ID, StatusRunning, nil, "", "", nil)
	if err != nil {
		return Command{}, err
	}
	_ = s.repo.AppendCommandEvent(ctx, reqCtx, running.ID, "command.running", payload)

	result := map[string]any{
		"handled":     true,
		"commandType": commandType,
		"executedAt":  time.Now().UTC().Format(time.RFC3339Nano),
	}
	resultBytes, _ := json.Marshal(result)
	finishedAt := time.Now().UTC()
	final, err := s.repo.SetStatus(ctx, reqCtx, running.ID, StatusSucceeded, resultBytes, "", "", &finishedAt)
	if err != nil {
		return Command{}, err
	}

	_ = s.repo.AppendCommandEvent(ctx, reqCtx, final.ID, "command.succeeded", resultBytes)
	_ = s.repo.AppendAuditEvent(ctx, reqCtx, final.ID, "command.execute", "allow", "stub_execute", resultBytes)

	return final, nil
}

func (s *Service) Get(ctx context.Context, reqCtx RequestContext, id string) (Command, error) {
	return s.repo.Get(ctx, reqCtx, id)
}

func (s *Service) List(ctx context.Context, params ListParams) (ListResult, error) {
	return s.repo.List(ctx, params)
}

func validateCreateRequest(commandType string, payload json.RawMessage) error {
	if strings.TrimSpace(commandType) == "" {
		return ErrInvalidCommandRequest
	}
	if len(payload) == 0 {
		return ErrInvalidCommandRequest
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return ErrInvalidCommandRequest
	}

	return nil
}

func hashRequest(commandType string, payload json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(commandType))
	h.Write([]byte("\n"))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
