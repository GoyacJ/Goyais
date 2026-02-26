package acp

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const sessionStoreVersion = 1

var nonSlugChars = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var nonSessionIDChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

type PersistedMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type PersistedSession struct {
	Version       int                `json:"version"`
	SessionID     string             `json:"sessionId"`
	CWD           string             `json:"cwd"`
	CurrentModeID string             `json:"currentModeId"`
	Messages      []PersistedMessage `json:"messages"`
}

func PersistSession(baseDir string, value PersistedSession) error {
	if strings.TrimSpace(value.SessionID) == "" {
		return errors.New("sessionId is required")
	}
	if strings.TrimSpace(value.CWD) == "" {
		return errors.New("cwd is required")
	}
	if value.Messages == nil {
		value.Messages = []PersistedMessage{}
	}
	if strings.TrimSpace(value.CurrentModeID) == "" {
		value.CurrentModeID = "default"
	}
	value.Version = sessionStoreVersion

	path := SessionFilePath(baseDir, value.CWD, value.SessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func LoadSession(baseDir string, cwd string, sessionID string) (PersistedSession, error) {
	path := SessionFilePath(baseDir, cwd, sessionID)
	body, err := os.ReadFile(path)
	if err != nil {
		return PersistedSession{}, err
	}
	value := PersistedSession{}
	if err := json.Unmarshal(body, &value); err != nil {
		return PersistedSession{}, err
	}
	if strings.TrimSpace(value.SessionID) != strings.TrimSpace(sessionID) {
		return PersistedSession{}, errors.New("session id mismatch")
	}
	if strings.TrimSpace(value.CWD) != strings.TrimSpace(cwd) {
		return PersistedSession{}, errors.New("cwd mismatch")
	}
	if value.Messages == nil {
		value.Messages = []PersistedMessage{}
	}
	if strings.TrimSpace(value.CurrentModeID) == "" {
		value.CurrentModeID = "default"
	}
	return value, nil
}

func SessionFilePath(baseDir string, cwd string, sessionID string) string {
	storeDir := SessionDirectory(baseDir, cwd)
	return filepath.Join(storeDir, sanitizeSessionID(sessionID)+".json")
}

func SessionDirectory(baseDir string, cwd string) string {
	root := strings.TrimSpace(baseDir)
	if root == "" {
		root = defaultACPBaseDir()
	}
	return filepath.Join(root, slugForPath(cwd), "acp-sessions")
}

func defaultACPBaseDir() string {
	if override := strings.TrimSpace(os.Getenv("GOYAIS_ACP_BASE_DIR")); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ".goyais"
	}
	return filepath.Join(home, ".goyais")
}

func slugForPath(cwd string) string {
	trimmed := strings.TrimSpace(cwd)
	if trimmed == "" {
		return "default"
	}
	return strings.Trim(nonSlugChars.ReplaceAllString(trimmed, "-"), "-")
}

func sanitizeSessionID(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return "session"
	}
	return strings.Trim(nonSessionIDChars.ReplaceAllString(trimmed, "_"), "_")
}
