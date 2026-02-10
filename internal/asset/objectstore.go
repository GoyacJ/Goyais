package asset

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type ObjectStore interface {
	Put(ctx context.Context, req command.RequestContext, hash string, data []byte, now time.Time) (string, error)
}

type NotImplementedStore struct {
	provider string
}

func NewObjectStore(provider, localRoot string) ObjectStore {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "local":
		return NewLocalStore(localRoot)
	case "minio", "s3":
		return &NotImplementedStore{provider: provider}
	default:
		return &NotImplementedStore{provider: provider}
	}
}

func (s *NotImplementedStore) Put(context.Context, command.RequestContext, string, []byte, time.Time) (string, error) {
	return "", fmt.Errorf("%w: object_store.provider=%s", ErrNotImplemented, s.provider)
}
