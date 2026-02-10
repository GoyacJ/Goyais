package asset

import (
	"context"
	"errors"
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

func (s *LocalStore) Get(_ context.Context, uri string) ([]byte, error) {
	absolute, err := s.resolvePath(uri)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return nil, fmt.Errorf("%w: read: %v", ErrObjectStoreFail, err)
	}
	return raw, nil
}

func (s *LocalStore) Delete(_ context.Context, uri string) error {
	absolute, err := s.resolvePath(uri)
	if err != nil {
		return err
	}
	if err := os.Remove(absolute); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: delete: %v", ErrObjectStoreFail, err)
	}
	return nil
}

func (s *LocalStore) Ping(_ context.Context) error {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("%w: root: %v", ErrObjectStoreFail, err)
	}
	return nil
}

func (s *LocalStore) Provider() string {
	return "local"
}

func (s *LocalStore) resolvePath(uri string) (string, error) {
	normalized := strings.TrimSpace(uri)
	if !strings.HasPrefix(normalized, "local://") {
		return "", fmt.Errorf("%w: invalid uri", ErrObjectStoreFail)
	}
	relative := strings.TrimPrefix(normalized, "local://")
	if strings.TrimSpace(relative) == "" {
		return "", fmt.Errorf("%w: empty uri path", ErrObjectStoreFail)
	}
	absolute := filepath.Clean(filepath.Join(s.root, filepath.FromSlash(relative)))
	root := filepath.Clean(s.root)
	if absolute != root && !strings.HasPrefix(absolute, root+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: invalid uri path", ErrObjectStoreFail)
	}
	return absolute, nil
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
