package httpapi

import (
	_ "embed"
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

var (
	//go:embed templates/models.default.json
	defaultModelCatalogTemplate []byte
)

const (
	workspaceModelCatalogRelativePath = ".goyais/model.json"
	defaultModelCatalogSource         = "embedded://models.default.json"
)

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

	filePath := filepath.Join(root.CatalogRoot, workspaceModelCatalogRelativePath)
	raw, revision, source, err := resolveWorkspaceModelCatalog(filePath)
	if err != nil {
		return ModelCatalogResponse{}, err
	}

	if !force {
		s.mu.RLock()
		cache, ok := s.modelCatalogCache[workspaceID]
		s.mu.RUnlock()
		if ok && cache.Revision == revision && cache.FilePath == source {
			return cache.Response, nil
		}
	}

	payload, err := parseModelCatalogPayload(raw, source)
	if err != nil {
		if source == defaultModelCatalogSource {
			return ModelCatalogResponse{}, err
		}
		raw, revision, source, err = loadEmbeddedModelCatalogData()
		if err != nil {
			return ModelCatalogResponse{}, err
		}
		payload, err = parseModelCatalogPayload(raw, source)
		if err != nil {
			return ModelCatalogResponse{}, err
		}
	}

	normalized := make([]ModelCatalogVendor, 0, len(payload.Vendors))
	for _, vendor := range payload.Vendors {
		if !isSupportedVendor(vendor.Name) {
			return ModelCatalogResponse{}, fmt.Errorf("unsupported vendor: %s", vendor.Name)
		}
		baseURL := strings.TrimSpace(vendor.BaseURL)
		if baseURL == "" {
			return ModelCatalogResponse{}, fmt.Errorf("vendor %s has empty base_url", vendor.Name)
		}
		if !isValidURLString(baseURL) {
			return ModelCatalogResponse{}, fmt.Errorf("vendor %s has invalid base_url", vendor.Name)
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
			Name:    vendor.Name,
			BaseURL: baseURL,
			Models:  models,
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
		Source:      source,
		Vendors:     normalized,
	}

	s.mu.Lock()
	s.modelCatalogCache[workspaceID] = modelCatalogCacheEntry{
		WorkspaceID: workspaceID,
		FilePath:    source,
		Revision:    revision,
		Response:    response,
	}
	s.mu.Unlock()
	return response, nil
}

func (s *AppState) resolveCatalogVendorBaseURL(workspaceID string, vendor ModelVendorName) string {
	response, err := s.LoadModelCatalog(workspaceID, false)
	if err != nil {
		return ""
	}
	for _, item := range response.Vendors {
		if item.Name == vendor {
			return strings.TrimSpace(item.BaseURL)
		}
	}
	return ""
}

func (s *AppState) defaultCatalogRoot(workspaceID string) string {
	workspace, exists := s.GetWorkspace(workspaceID)
	if exists && workspace.Mode == WorkspaceModeRemote {
		return filepath.Join("data", "workspaces", workspaceID, "config")
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return home
	}
	return filepath.Join(".", "data", "local")
}

func loadDefaultModelCatalogTemplate(now string) (modelCatalogFile, error) {
	payload := modelCatalogFile{}
	if err := json.Unmarshal(defaultModelCatalogTemplate, &payload); err != nil {
		return modelCatalogFile{}, fmt.Errorf("invalid embedded models.default.json: %w", err)
	}
	payload.UpdatedAt = now
	return payload, nil
}

func resolveWorkspaceModelCatalog(path string) ([]byte, int64, string, error) {
	info, err := os.Stat(path)
	if err == nil {
		raw, readErr := os.ReadFile(path)
		if readErr == nil {
			return raw, info.ModTime().UnixNano(), path, nil
		}
	}

	return loadEmbeddedModelCatalogData()
}

func loadEmbeddedModelCatalogData() ([]byte, int64, string, error) {
	payload, loadErr := loadDefaultModelCatalogTemplate(nowUTC())
	if loadErr != nil {
		return nil, 0, "", loadErr
	}
	encoded, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return nil, 0, "", marshalErr
	}
	return encoded, 0, defaultModelCatalogSource, nil
}

func parseModelCatalogPayload(raw []byte, source string) (modelCatalogFile, error) {
	payload := modelCatalogFile{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return modelCatalogFile{}, fmt.Errorf("invalid model catalog (%s): %w", source, err)
	}
	if len(payload.Vendors) == 0 {
		return modelCatalogFile{}, fmt.Errorf("model catalog (%s) vendors cannot be empty", source)
	}
	return payload, nil
}

func isSupportedVendor(vendor ModelVendorName) bool {
	for _, item := range supportedModelVendors {
		if item == vendor {
			return true
		}
	}
	return false
}
