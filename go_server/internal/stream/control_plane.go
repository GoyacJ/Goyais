package stream

import (
	"context"
	"encoding/json"
)

type ControlPlane interface {
	EnsurePath(ctx context.Context, streamPath string, source string, state json.RawMessage) error
	PatchPathAuth(ctx context.Context, streamPath string, authRule map[string]any) error
	DeletePath(ctx context.Context, streamPath string) error
	KickPath(ctx context.Context, streamPath string) error
}
