package asset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goyais/internal/command"
)

type LocalStore struct {
	root string
}

func NewLocalStore(root string) *LocalStore {
	if strings.TrimSpace(root) == "" {
		root = "./data/objects"
	}
	return &LocalStore{root: root}
}

func (s *LocalStore) Put(_ context.Context, req command.RequestContext, hash string, data []byte, now time.Time) (string, error) {
	if strings.TrimSpace(hash) == "" {
		return "", fmt.Errorf("%w: empty hash", ErrObjectStoreFail)
	}
	datePath := now.UTC().Format("2006/01/02")
	relative := filepath.ToSlash(filepath.Join(safePath(req.TenantID), safePath(req.WorkspaceID), datePath, strings.ToLower(hash)))
	absolute := filepath.Join(s.root, filepath.FromSlash(relative))

	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return "", fmt.Errorf("%w: mkdir: %v", ErrObjectStoreFail, err)
	}
	if err := os.WriteFile(absolute, data, 0o644); err != nil {
		return "", fmt.Errorf("%w: write: %v", ErrObjectStoreFail, err)
	}
	return "local://" + relative, nil
}

func safePath(value string) string {
	clean := strings.TrimSpace(value)
	clean = strings.ReplaceAll(clean, "..", "")
	clean = strings.ReplaceAll(clean, string(filepath.Separator), "_")
	if clean == "" {
		return "unknown"
	}
	return clean
}
