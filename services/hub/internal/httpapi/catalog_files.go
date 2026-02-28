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

type modelCatalogParseMeta struct {
	AutoFilled bool
}

type modelCatalogLoadMeta struct {
	Source            string
	Revision          int64
	FromCache         bool
	FallbackUsed      bool
	FallbackReason    string
	FallbackError     string
	AutoFilled        bool
	AutoFillWriteback bool
	AutoFillWriteErr  string
}

type resolvedCatalogSource struct {
	Raw               []byte
	Revision          int64
	Source            string
	WorkspaceFilePath string
	FromWorkspaceFile bool
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
				traceID := GenerateTraceID()
				response, meta, loadErr := s.loadModelCatalogDetailed(workspace.ID, false)
				if loadErr == nil {
					meta.Source = response.Source
					meta.Revision = response.Revision
				}
				s.recordModelCatalogReloadAudit(workspace.ID, "scheduled", meta, loadErr, "system", traceID)
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
	response, _, err := s.loadModelCatalogDetailed(workspaceID, force)
	return response, err
}

func (s *AppState) loadModelCatalogDetailed(workspaceID string, force bool) (ModelCatalogResponse, modelCatalogLoadMeta, error) {
	meta := modelCatalogLoadMeta{}
	root, err := s.GetCatalogRoot(workspaceID)
	if err != nil {
		return ModelCatalogResponse{}, meta, err
	}

	filePath := filepath.Join(root.CatalogRoot, workspaceModelCatalogRelativePath)
	resolved, err := resolveWorkspaceModelCatalog(filePath)
	if err != nil {
		return ModelCatalogResponse{}, meta, err
	}
	meta.Source = resolved.Source
	meta.Revision = resolved.Revision

	if !force {
		s.mu.RLock()
		cache, ok := s.modelCatalogCache[workspaceID]
		s.mu.RUnlock()
		if ok && cache.Revision == resolved.Revision && cache.FilePath == resolved.Source {
			meta.FromCache = true
			meta.Source = cache.Response.Source
			meta.Revision = cache.Response.Revision
			return cache.Response, meta, nil
		}
	}

	payload, parseMeta, parseErr := parseModelCatalogPayload(resolved.Raw, resolved.Source, resolved.FromWorkspaceFile)
	if parseErr != nil {
		if resolved.Source == defaultModelCatalogSource {
			return ModelCatalogResponse{}, meta, parseErr
		}
		meta.FallbackUsed = true
		meta.FallbackReason = "parse_failed"
		meta.FallbackError = parseErr.Error()
		resolved, payload, err = fallbackToEmbeddedCatalog("parse_failed")
		if err != nil {
			return ModelCatalogResponse{}, meta, err
		}
	}

	if resolved.FromWorkspaceFile && parseMeta.AutoFilled {
		meta.AutoFilled = true
		encoded, marshalErr := marshalModelCatalogPayload(payload)
		if marshalErr != nil {
			meta.FallbackUsed = true
			meta.FallbackReason = "autofill_marshal_failed"
			meta.FallbackError = marshalErr.Error()
			resolved, payload, err = fallbackToEmbeddedCatalog("autofill_marshal_failed")
			if err != nil {
				return ModelCatalogResponse{}, meta, err
			}
		} else {
			writeErr := os.WriteFile(resolved.WorkspaceFilePath, encoded, 0o644)
			if writeErr != nil {
				meta.AutoFillWriteErr = writeErr.Error()
				meta.FallbackUsed = true
				meta.FallbackReason = "autofill_write_failed"
				meta.FallbackError = writeErr.Error()
				resolved, payload, err = fallbackToEmbeddedCatalog("autofill_write_failed")
				if err != nil {
					return ModelCatalogResponse{}, meta, err
				}
			} else {
				meta.AutoFillWriteback = true
				resolved.Raw = encoded
				resolved.Source = resolved.WorkspaceFilePath
				if info, statErr := os.Stat(resolved.WorkspaceFilePath); statErr == nil {
					resolved.Revision = info.ModTime().UnixNano()
				}
			}
		}
	}

	updatedAt := strings.TrimSpace(payload.UpdatedAt)
	if updatedAt == "" {
		updatedAt = nowUTC()
	}
	response := ModelCatalogResponse{
		WorkspaceID: workspaceID,
		Revision:    resolved.Revision,
		UpdatedAt:   updatedAt,
		Source:      resolved.Source,
		Vendors:     payload.Vendors,
	}

	s.mu.Lock()
	s.modelCatalogCache[workspaceID] = modelCatalogCacheEntry{
		WorkspaceID: workspaceID,
		FilePath:    response.Source,
		Revision:    response.Revision,
		Response:    response,
	}
	s.mu.Unlock()

	meta.Source = response.Source
	meta.Revision = response.Revision
	return response, meta, nil
}

func fallbackToEmbeddedCatalog(reason string) (resolvedCatalogSource, modelCatalogFile, error) {
	resolved, err := loadEmbeddedModelCatalogData()
	if err != nil {
		return resolvedCatalogSource{}, modelCatalogFile{}, fmt.Errorf("fallback to embedded catalog failed (%s): %w", reason, err)
	}
	payload, _, parseErr := parseModelCatalogPayload(resolved.Raw, resolved.Source, false)
	if parseErr != nil {
		return resolvedCatalogSource{}, modelCatalogFile{}, parseErr
	}
	return resolved, payload, nil
}

func (s *AppState) resolveCatalogVendor(workspaceID string, vendor ModelVendorName) (ModelCatalogVendor, bool) {
	response, err := s.LoadModelCatalog(workspaceID, false)
	if err != nil {
		return ModelCatalogVendor{}, false
	}
	for _, item := range response.Vendors {
		if item.Name == vendor {
			return item, true
		}
	}
	return ModelCatalogVendor{}, false
}

func (s *AppState) resolveCatalogVendorBaseURL(workspaceID string, vendor ModelVendorName) string {
	item, ok := s.resolveCatalogVendor(workspaceID, vendor)
	if !ok {
		return ""
	}
	return strings.TrimSpace(item.BaseURL)
}

func (s *AppState) resolveWorkspaceDefaultModelID(workspaceID string) string {
	response, err := s.LoadModelCatalog(workspaceID, false)
	if err != nil {
		return ""
	}
	return resolveDefaultModelIDFromCatalog(response.Vendors)
}

func resolveDefaultModelIDFromCatalog(vendors []ModelCatalogVendor) string {
	firstEnabled := ""
	for _, vendor := range vendors {
		for _, model := range vendor.Models {
			if !model.Enabled {
				continue
			}
			if firstEnabled == "" {
				firstEnabled = strings.TrimSpace(model.ID)
			}
			if strings.Contains(strings.ToLower(strings.TrimSpace(model.Label)), "(default)") {
				return strings.TrimSpace(model.ID)
			}
		}
	}
	return firstEnabled
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
	payload.Version = firstNonEmpty(strings.TrimSpace(payload.Version), "1")
	return payload, nil
}

func resolveWorkspaceModelCatalog(path string) (resolvedCatalogSource, error) {
	info, err := os.Stat(path)
	if err == nil {
		raw, readErr := os.ReadFile(path)
		if readErr == nil {
			return resolvedCatalogSource{
				Raw:               raw,
				Revision:          info.ModTime().UnixNano(),
				Source:            path,
				WorkspaceFilePath: path,
				FromWorkspaceFile: true,
			}, nil
		}
	}

	return loadEmbeddedModelCatalogData()
}

func loadEmbeddedModelCatalogData() (resolvedCatalogSource, error) {
	payload, loadErr := loadDefaultModelCatalogTemplate(nowUTC())
	if loadErr != nil {
		return resolvedCatalogSource{}, loadErr
	}
	encoded, marshalErr := marshalModelCatalogPayload(payload)
	if marshalErr != nil {
		return resolvedCatalogSource{}, marshalErr
	}
	return resolvedCatalogSource{
		Raw:               encoded,
		Revision:          0,
		Source:            defaultModelCatalogSource,
		WorkspaceFilePath: "",
		FromWorkspaceFile: false,
	}, nil
}

func parseModelCatalogPayload(raw []byte, source string, allowAutoFill bool) (modelCatalogFile, modelCatalogParseMeta, error) {
	payload := modelCatalogFile{}
	meta := modelCatalogParseMeta{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return modelCatalogFile{}, meta, fmt.Errorf("invalid model catalog (%s): %w", source, err)
	}

	payload.Version = firstNonEmpty(strings.TrimSpace(payload.Version), "1")
	if len(payload.Vendors) == 0 {
		return modelCatalogFile{}, meta, fmt.Errorf("model catalog (%s) vendors cannot be empty", source)
	}

	normalized := make([]ModelCatalogVendor, 0, len(payload.Vendors))
	for _, vendor := range payload.Vendors {
		vendorName := ModelVendorName(strings.TrimSpace(string(vendor.Name)))
		if !isSupportedVendor(vendorName) {
			return modelCatalogFile{}, meta, fmt.Errorf("unsupported vendor: %s", vendor.Name)
		}

		baseURL := strings.TrimSpace(vendor.BaseURL)
		if baseURL == "" {
			return modelCatalogFile{}, meta, fmt.Errorf("vendor %s has empty base_url", vendorName)
		}
		if !isValidURLString(baseURL) {
			return modelCatalogFile{}, meta, fmt.Errorf("vendor %s has invalid base_url", vendorName)
		}

		homepage := strings.TrimSpace(vendor.Homepage)
		if homepage != "" && !isValidURLString(homepage) {
			return modelCatalogFile{}, meta, fmt.Errorf("vendor %s has invalid homepage", vendorName)
		}
		docs := strings.TrimSpace(vendor.Docs)
		if docs != "" && !isValidURLString(docs) {
			return modelCatalogFile{}, meta, fmt.Errorf("vendor %s has invalid docs", vendorName)
		}

		baseURLs := map[string]string{}
		for key, value := range vendor.BaseURLs {
			normalizedKey := strings.TrimSpace(key)
			normalizedValue := strings.TrimSpace(value)
			if normalizedKey == "" || normalizedValue == "" {
				continue
			}
			if !isValidURLString(normalizedValue) {
				return modelCatalogFile{}, meta, fmt.Errorf("vendor %s base_urls[%s] is invalid", vendorName, normalizedKey)
			}
			baseURLs[normalizedKey] = normalizedValue
		}

		auth, authAutoFilled, authErr := normalizeVendorAuth(vendorName, vendor.Auth, allowAutoFill)
		if authErr != nil {
			return modelCatalogFile{}, meta, authErr
		}
		if authAutoFilled {
			meta.AutoFilled = true
		}

		models := make([]ModelCatalogModel, 0, len(vendor.Models))
		for _, model := range vendor.Models {
			modelID := strings.TrimSpace(model.ID)
			if modelID == "" {
				return modelCatalogFile{}, meta, fmt.Errorf("vendor %s has empty model id", vendorName)
			}
			label := strings.TrimSpace(model.Label)
			if label == "" {
				label = modelID
				if allowAutoFill {
					meta.AutoFilled = true
				}
			}
			models = append(models, ModelCatalogModel{ID: modelID, Label: label, Enabled: model.Enabled})
		}
		if len(models) == 0 {
			return modelCatalogFile{}, meta, fmt.Errorf("vendor %s models cannot be empty", vendorName)
		}

		notes := make([]string, 0, len(vendor.Notes))
		for _, note := range vendor.Notes {
			normalizedNote := strings.TrimSpace(note)
			if normalizedNote == "" {
				continue
			}
			notes = append(notes, normalizedNote)
		}

		normalizedVendor := ModelCatalogVendor{
			Name:     vendorName,
			Homepage: homepage,
			Docs:     docs,
			BaseURL:  baseURL,
			Auth:     auth,
			Models:   models,
			Notes:    notes,
		}
		if len(baseURLs) > 0 {
			normalizedVendor.BaseURLs = baseURLs
		}
		normalized = append(normalized, normalizedVendor)
	}

	payload.Vendors = normalized
	if strings.TrimSpace(payload.UpdatedAt) == "" {
		payload.UpdatedAt = nowUTC()
	}
	return payload, meta, nil
}

func normalizeVendorAuth(vendor ModelVendorName, input ModelCatalogVendorAuth, allowAutoFill bool) (ModelCatalogVendorAuth, bool, error) {
	auth := ModelCatalogVendorAuth{
		Type:      strings.TrimSpace(input.Type),
		Header:    strings.TrimSpace(input.Header),
		Scheme:    strings.TrimSpace(input.Scheme),
		APIKeyEnv: strings.TrimSpace(input.APIKeyEnv),
	}
	defaults := defaultVendorAuth(vendor)
	autoFilled := false

	if auth.Type == "" {
		if !allowAutoFill {
			return ModelCatalogVendorAuth{}, false, fmt.Errorf("vendor %s auth is required", vendor)
		}
		auth = defaults
		autoFilled = true
	}

	switch auth.Type {
	case "none":
		auth.Header = ""
		auth.Scheme = ""
		auth.APIKeyEnv = ""
		return auth, autoFilled, nil
	case "http_bearer":
		if auth.Header == "" {
			if !allowAutoFill {
				return ModelCatalogVendorAuth{}, false, fmt.Errorf("vendor %s auth.header is required", vendor)
			}
			auth.Header = defaults.Header
			autoFilled = true
		}
		if auth.Scheme == "" {
			if !allowAutoFill {
				return ModelCatalogVendorAuth{}, false, fmt.Errorf("vendor %s auth.scheme is required", vendor)
			}
			auth.Scheme = defaults.Scheme
			autoFilled = true
		}
		if auth.APIKeyEnv == "" {
			if !allowAutoFill {
				return ModelCatalogVendorAuth{}, false, fmt.Errorf("vendor %s auth.api_key_env is required", vendor)
			}
			auth.APIKeyEnv = defaults.APIKeyEnv
			autoFilled = true
		}
		return auth, autoFilled, nil
	case "api_key_header":
		if auth.Header == "" {
			if !allowAutoFill {
				return ModelCatalogVendorAuth{}, false, fmt.Errorf("vendor %s auth.header is required", vendor)
			}
			auth.Header = defaults.Header
			autoFilled = true
		}
		if auth.APIKeyEnv == "" {
			if !allowAutoFill {
				return ModelCatalogVendorAuth{}, false, fmt.Errorf("vendor %s auth.api_key_env is required", vendor)
			}
			auth.APIKeyEnv = defaults.APIKeyEnv
			autoFilled = true
		}
		auth.Scheme = ""
		return auth, autoFilled, nil
	default:
		return ModelCatalogVendorAuth{}, false, fmt.Errorf("vendor %s auth.type is invalid", vendor)
	}
}

func defaultVendorAuth(vendor ModelVendorName) ModelCatalogVendorAuth {
	switch vendor {
	case ModelVendorDeepSeek:
		return ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer", APIKeyEnv: "DEEPSEEK_API_KEY"}
	case ModelVendorGoogle:
		return ModelCatalogVendorAuth{Type: "api_key_header", Header: "x-goog-api-key", APIKeyEnv: "GEMINI_API_KEY"}
	case ModelVendorQwen:
		return ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer", APIKeyEnv: "DASHSCOPE_API_KEY"}
	case ModelVendorDoubao:
		return ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer", APIKeyEnv: "ARK_API_KEY"}
	case ModelVendorZhipu:
		return ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer", APIKeyEnv: "ZHIPU_API_KEY"}
	case ModelVendorMiniMax:
		return ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer", APIKeyEnv: "MINIMAX_API_KEY"}
	case ModelVendorLocal:
		return ModelCatalogVendorAuth{Type: "none"}
	default:
		return ModelCatalogVendorAuth{Type: "http_bearer", Header: "Authorization", Scheme: "Bearer", APIKeyEnv: "OPENAI_API_KEY"}
	}
}

func marshalModelCatalogPayload(payload modelCatalogFile) ([]byte, error) {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(encoded) == 0 || encoded[len(encoded)-1] != '\n' {
		encoded = append(encoded, '\n')
	}
	return encoded, nil
}

func isSupportedVendor(vendor ModelVendorName) bool {
	for _, item := range supportedModelVendors {
		if item == vendor {
			return true
		}
	}
	return false
}

func normalizeCatalogReloadTrigger(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "page_open":
		return "page_open"
	case "scheduled":
		return "scheduled"
	default:
		return "manual"
	}
}

func (s *AppState) recordModelCatalogReloadAudit(workspaceID string, trigger string, meta modelCatalogLoadMeta, loadErr error, actor string, traceID string) {
	normalizedTrigger := normalizeCatalogReloadTrigger(trigger)
	if normalizedTrigger == "scheduled" && meta.FromCache && loadErr == nil {
		return
	}
	details := map[string]any{
		"operation": "model_catalog.reload",
		"trigger":   normalizedTrigger,
		"source":    meta.Source,
		"revision":  meta.Revision,
		"cached":    meta.FromCache,
	}
	s.appendModelCatalogAudit(workspaceID, actor, traceID, "requested", "success", details)

	if loadErr != nil {
		details["error"] = loadErr.Error()
		details["reason"] = firstNonEmpty(meta.FallbackReason, "load_failed")
		s.appendModelCatalogAudit(workspaceID, actor, traceID, "fallback_or_failed", "failed", details)
		return
	}

	details["fallback_used"] = meta.FallbackUsed
	details["fallback_reason"] = meta.FallbackReason
	details["fallback_error"] = meta.FallbackError
	details["autofilled"] = meta.AutoFilled
	details["autofill_writeback"] = meta.AutoFillWriteback
	details["autofill_write_error"] = meta.AutoFillWriteErr
	s.appendModelCatalogAudit(workspaceID, actor, traceID, "apply", "success", details)
	if meta.FallbackUsed {
		s.appendModelCatalogAudit(workspaceID, actor, traceID, "fallback_or_failed", "success", details)
	}
}

func (s *AppState) appendModelCatalogAudit(workspaceID string, actor string, traceID string, stage string, result string, details map[string]any) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return
	}
	normalizedActor := firstNonEmpty(strings.TrimSpace(actor), "system")
	normalizedTraceID := firstNonEmpty(strings.TrimSpace(traceID), GenerateTraceID())
	action := "model_catalog.reload"
	if strings.TrimSpace(stage) != "" {
		action = action + "." + strings.TrimSpace(stage)
	}
	s.AppendAudit(AdminAuditEvent{
		Actor:    normalizedActor,
		Action:   action,
		Resource: normalizedWorkspaceID,
		Result:   result,
		TraceID:  normalizedTraceID,
	})
	if s.authz != nil {
		copyDetails := map[string]any{}
		for key, value := range details {
			copyDetails[key] = value
		}
		copyDetails["stage"] = stage
		_ = s.authz.appendAudit(normalizedWorkspaceID, normalizedActor, action, "workspace", normalizedWorkspaceID, result, copyDetails, normalizedTraceID)
	}
}
