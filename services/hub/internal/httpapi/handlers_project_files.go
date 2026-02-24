package httpapi

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxProjectFilePreviewBytes = 512 * 1024

func ProjectFilesHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		projectID := strings.TrimSpace(r.PathValue("project_id"))
		project, exists, err := getProjectFromStore(state, projectID)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{"project_id": projectID})
			return
		}
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": projectID})
			return
		}
		_, authErr := authorizeAction(
			state,
			r,
			project.WorkspaceID,
			"project.read",
			authorizationResource{WorkspaceID: project.WorkspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		relativePath := strings.TrimSpace(r.URL.Query().Get("path"))
		depth := 2
		if rawDepth := strings.TrimSpace(r.URL.Query().Get("depth")); rawDepth != "" {
			parsedDepth, parseErr := strconv.Atoi(rawDepth)
			if parseErr != nil || parsedDepth < 1 || parsedDepth > 6 {
				WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "depth must be an integer between 1 and 6", map[string]any{})
				return
			}
			depth = parsedDepth
		}

		targetPath, relativeTarget, resolveErr := resolveProjectPath(project.RepoPath, relativePath)
		if resolveErr != nil {
			WriteStandardError(w, r, http.StatusBadRequest, "PATH_OUT_OF_PROJECT", "Path must stay within project root", map[string]any{
				"path": relativePath,
			})
			return
		}

		items, listErr := listProjectEntries(project.RepoPath, targetPath, relativeTarget, depth)
		if listErr != nil {
			if os.IsNotExist(listErr) {
				WriteStandardError(w, r, http.StatusNotFound, "FILE_NOT_FOUND", "Path does not exist", map[string]any{"path": relativePath})
				return
			}
			WriteStandardError(w, r, http.StatusInternalServerError, "FILE_LIST_FAILED", "Failed to list project files", map[string]any{
				"path":  relativePath,
				"error": listErr.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, items)
	}
}

func ProjectFileContentHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteStandardError(w, r, http.StatusNotImplemented, "INTERNAL_NOT_IMPLEMENTED", "Route is not implemented yet", map[string]any{
				"method": r.Method, "path": r.URL.Path,
			})
			return
		}
		projectID := strings.TrimSpace(r.PathValue("project_id"))
		project, exists, err := getProjectFromStore(state, projectID)
		if err != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "PROJECT_READ_FAILED", "Failed to read project", map[string]any{"project_id": projectID})
			return
		}
		if !exists {
			WriteStandardError(w, r, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project does not exist", map[string]any{"project_id": projectID})
			return
		}
		_, authErr := authorizeAction(
			state,
			r,
			project.WorkspaceID,
			"project.read",
			authorizationResource{WorkspaceID: project.WorkspaceID},
			authorizationContext{OperationType: "read"},
		)
		if authErr != nil {
			authErr.write(w, r)
			return
		}

		relativePath := strings.TrimSpace(r.URL.Query().Get("path"))
		if relativePath == "" {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "path is required", map[string]any{})
			return
		}
		targetPath, normalizedPath, resolveErr := resolveProjectPath(project.RepoPath, relativePath)
		if resolveErr != nil {
			WriteStandardError(w, r, http.StatusBadRequest, "PATH_OUT_OF_PROJECT", "Path must stay within project root", map[string]any{
				"path": relativePath,
			})
			return
		}
		info, statErr := os.Stat(targetPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				WriteStandardError(w, r, http.StatusNotFound, "FILE_NOT_FOUND", "Path does not exist", map[string]any{"path": relativePath})
				return
			}
			WriteStandardError(w, r, http.StatusInternalServerError, "FILE_READ_FAILED", "Failed to read file", map[string]any{"path": relativePath})
			return
		}
		if info.IsDir() {
			WriteStandardError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "path must point to a file", map[string]any{"path": relativePath})
			return
		}
		if info.Size() > maxProjectFilePreviewBytes {
			WriteStandardError(w, r, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "File preview exceeds limit", map[string]any{
				"path":      relativePath,
				"max_bytes": maxProjectFilePreviewBytes,
			})
			return
		}
		raw, readErr := os.ReadFile(targetPath)
		if readErr != nil {
			WriteStandardError(w, r, http.StatusInternalServerError, "FILE_READ_FAILED", "Failed to read file", map[string]any{
				"path":  relativePath,
				"error": readErr.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, ProjectFileContentResponse{
			Path:    normalizedPath,
			Content: string(raw),
		})
	}
}

func resolveProjectPath(projectRoot string, relativePath string) (string, string, error) {
	rootAbs, err := filepath.Abs(strings.TrimSpace(projectRoot))
	if err != nil {
		return "", "", err
	}
	targetRelative := strings.TrimSpace(relativePath)
	if targetRelative == "" {
		targetRelative = "."
	}
	targetClean := filepath.Clean(targetRelative)
	targetAbs := filepath.Join(rootAbs, targetClean)
	targetAbs, err = filepath.Abs(targetAbs)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", "", fs.ErrPermission
	}
	normalizedRel := filepath.ToSlash(strings.TrimPrefix(rel, "./"))
	if normalizedRel == "." {
		normalizedRel = ""
	}
	return targetAbs, normalizedRel, nil
}

func listProjectEntries(projectRoot string, targetAbs string, targetRel string, depth int) ([]ProjectFileEntry, error) {
	info, err := os.Stat(targetAbs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []ProjectFileEntry{makeProjectFileEntry(targetRel, info)}, nil
	}

	base := targetRel
	if base == "" {
		base = "."
	}
	result := make([]ProjectFileEntry, 0)
	var walk func(currentAbs string, currentRel string, remainingDepth int) error
	walk = func(currentAbs string, currentRel string, remainingDepth int) error {
		entries, readErr := os.ReadDir(currentAbs)
		if readErr != nil {
			return readErr
		}
		for _, entry := range entries {
			abs := filepath.Join(currentAbs, entry.Name())
			rel := filepath.ToSlash(filepath.Join(currentRel, entry.Name()))
			info, statErr := entry.Info()
			if statErr != nil {
				return statErr
			}
			result = append(result, makeProjectFileEntry(rel, info))
			if entry.IsDir() && remainingDepth > 1 {
				if err := walk(abs, rel, remainingDepth-1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(targetAbs, base, depth); err != nil {
		return nil, err
	}
	return result, nil
}

func makeProjectFileEntry(path string, info fs.FileInfo) ProjectFileEntry {
	entryType := "file"
	size := info.Size()
	if info.IsDir() {
		entryType = "directory"
		size = 0
	}
	return ProjectFileEntry{
		Path:  filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(path), "./")),
		Type:  entryType,
		Size:  size,
		MTime: info.ModTime().UTC().Format(time.RFC3339),
	}
}
