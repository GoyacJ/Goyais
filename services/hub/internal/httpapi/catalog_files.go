package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type modelCatalogFile struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Vendors   []ModelCatalogVendor `json:"vendors"`
}

type modelCatalogCacheEntry struct {
	WorkspaceID string
	FilePath    string
	Revision    int64
	Response    ModelCatalogResponse
}

func (s *AppState) startCatalogWatcher() {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for _, workspace := range s.ListWorkspaces() {
				_, _ = s.LoadModelCatalog(workspace.ID, false)
			}
		}
	}()
}

func (s *AppState) SetCatalogRoot(workspaceID string, root string) (CatalogRootResponse, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return CatalogRootResponse{}, errors.New("workspace_id is required")
	}
	resolvedRoot := strings.TrimSpace(root)
	if resolvedRoot == "" {
		return CatalogRootResponse{}, errors.New("catalog_root is required")
	}
	resolvedRoot = filepath.Clean(resolvedRoot)

	now := time.Now().UTC().Format(time.RFC3339)
	response := CatalogRootResponse{
		WorkspaceID: workspaceID,
		CatalogRoot: resolvedRoot,
		UpdatedAt:   now,
	}

	if s.authz != nil {
		if err := s.authz.upsertCatalogRoot(response); err != nil {
			return CatalogRootResponse{}, err
		}
	}

	s.mu.Lock()
	s.workspaceCatalogRoots[workspaceID] = response
	delete(s.modelCatalogCache, workspaceID)
	s.mu.Unlock()
	return response, nil
}

func (s *AppState) GetCatalogRoot(workspaceID string) (CatalogRootResponse, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return CatalogRootResponse{}, errors.New("workspace_id is required")
	}

	s.mu.RLock()
	cached, ok := s.workspaceCatalogRoots[workspaceID]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}

	if s.authz != nil {
		persisted, exists, err := s.authz.getCatalogRoot(workspaceID)
		if err != nil {
			return CatalogRootResponse{}, err
		}
		if exists {
			s.mu.Lock()
			s.workspaceCatalogRoots[workspaceID] = persisted
			s.mu.Unlock()
			return persisted, nil
		}
	}

	root := s.defaultCatalogRoot(workspaceID)
	fallback := CatalogRootResponse{
		WorkspaceID: workspaceID,
		CatalogRoot: root,
		UpdatedAt:   nowUTC(),
	}
	s.mu.Lock()
	s.workspaceCatalogRoots[workspaceID] = fallback
	s.mu.Unlock()
	return fallback, nil
}

func (s *AppState) LoadModelCatalog(workspaceID string, force bool) (ModelCatalogResponse, error) {
	root, err := s.GetCatalogRoot(workspaceID)
	if err != nil {
		return ModelCatalogResponse{}, err
	}

	filePath := filepath.Join(root.CatalogRoot, "goyais", "catalog", "models.json")
	if err := ensureCatalogFile(filePath); err != nil {
		return ModelCatalogResponse{}, err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return ModelCatalogResponse{}, err
	}
	revision := info.ModTime().UnixNano()

	if !force {
		s.mu.RLock()
		cache, ok := s.modelCatalogCache[workspaceID]
		s.mu.RUnlock()
		if ok && cache.Revision == revision && cache.FilePath == filePath {
			return cache.Response, nil
		}
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return ModelCatalogResponse{}, err
	}
	payload := modelCatalogFile{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ModelCatalogResponse{}, fmt.Errorf("invalid models.json: %w", err)
	}
	if len(payload.Vendors) == 0 {
		return ModelCatalogResponse{}, errors.New("models.json vendors cannot be empty")
	}

	normalized := make([]ModelCatalogVendor, 0, len(payload.Vendors))
	for _, vendor := range payload.Vendors {
		if !isSupportedVendor(vendor.Name) {
			return ModelCatalogResponse{}, fmt.Errorf("unsupported vendor: %s", vendor.Name)
		}
		models := make([]ModelCatalogModel, 0, len(vendor.Models))
		for _, model := range vendor.Models {
			if strings.TrimSpace(model.ID) == "" {
				return ModelCatalogResponse{}, fmt.Errorf("vendor %s has empty model id", vendor.Name)
			}
			item := model
			item.ID = strings.TrimSpace(item.ID)
			item.Label = firstNonEmpty(strings.TrimSpace(item.Label), item.ID)
			models = append(models, item)
		}
		normalized = append(normalized, ModelCatalogVendor{
			Name:   vendor.Name,
			Models: models,
		})
	}

	updatedAt := strings.TrimSpace(payload.UpdatedAt)
	if updatedAt == "" {
		updatedAt = nowUTC()
	}
	response := ModelCatalogResponse{
		WorkspaceID: workspaceID,
		Revision:    revision,
		UpdatedAt:   updatedAt,
		Source:      filePath,
		Vendors:     normalized,
	}

	s.mu.Lock()
	s.modelCatalogCache[workspaceID] = modelCatalogCacheEntry{
		WorkspaceID: workspaceID,
		FilePath:    filePath,
		Revision:    revision,
		Response:    response,
	}
	s.mu.Unlock()
	return response, nil
}

func (s *AppState) defaultCatalogRoot(workspaceID string) string {
	workspace, exists := s.GetWorkspace(workspaceID)
	if exists && workspace.Mode == WorkspaceModeRemote {
		return filepath.Join("data", "workspaces", workspaceID, "config")
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".goyais")
	}
	return filepath.Join(".", "data", "local")
}

func ensureCatalogFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	now := nowUTC()
	payload := modelCatalogFile{
		Version:   "1",
		UpdatedAt: now,
		Vendors: []ModelCatalogVendor{
			{Name: ModelVendorOpenAI, Models: []ModelCatalogModel{{ID: "gpt-4.1", Label: "GPT-4.1", Enabled: true}}},
			{Name: ModelVendorGoogle, Models: []ModelCatalogModel{{ID: "gemini-2.0-flash", Label: "Gemini 2.0 Flash", Enabled: true}}},
			{Name: ModelVendorQwen, Models: []ModelCatalogModel{{ID: "qwen-max", Label: "Qwen Max", Enabled: true}}},
			{Name: ModelVendorDoubao, Models: []ModelCatalogModel{{ID: "doubao-pro-32k", Label: "Doubao Pro 32k", Enabled: true}}},
			{Name: ModelVendorZhipu, Models: []ModelCatalogModel{{ID: "glm-4-plus", Label: "GLM-4-Plus", Enabled: true}}},
			{Name: ModelVendorMiniMax, Models: []ModelCatalogModel{{ID: "MiniMax-Text-01", Label: "MiniMax Text 01", Enabled: true}}},
			{Name: ModelVendorLocal, Models: []ModelCatalogModel{{ID: "llama3.1:8b", Label: "Llama 3.1 8B", Enabled: true}}},
		},
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o644)
}

func isSupportedVendor(vendor ModelVendorName) bool {
	for _, item := range supportedModelVendors {
		if item == vendor {
			return true
		}
	}
	return false
}
