package httpapi

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
)

func listProjectsFromStore(state *AppState, workspaceID string) ([]Project, error) {
	if state.authz != nil {
		items, err := state.authz.listProjects(workspaceID)
		if err != nil {
			return nil, err
		}
		state.mu.Lock()
		for _, item := range items {
			state.projects[item.ID] = item
		}
		state.mu.Unlock()
		return items, nil
	}

	state.mu.RLock()
	items := make([]Project, 0)
	for _, item := range state.projects {
		if workspaceID != "" && item.WorkspaceID != workspaceID {
			continue
		}
		items = append(items, item)
	}
	state.mu.RUnlock()
	return items, nil
}

func getProjectFromStore(state *AppState, projectID string) (Project, bool, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return Project{}, false, nil
	}
	if state.authz != nil {
		item, exists, err := state.authz.getProject(projectID)
		if err != nil {
			return Project{}, false, err
		}
		if exists {
			state.mu.Lock()
			state.projects[item.ID] = item
			state.mu.Unlock()
		}
		return item, exists, nil
	}
	state.mu.RLock()
	item, exists := state.projects[projectID]
	state.mu.RUnlock()
	return item, exists, nil
}

func saveProjectToStore(state *AppState, input Project) (Project, error) {
	item := input
	var err error
	if state.authz != nil {
		item, err = state.authz.upsertProject(input)
		if err != nil {
			return Project{}, err
		}
	}
	state.mu.Lock()
	state.projects[item.ID] = item
	state.mu.Unlock()
	return item, nil
}

func deleteProjectFromStore(state *AppState, projectID string) (Project, error) {
	project, exists, err := getProjectFromStore(state, projectID)
	if err != nil {
		return Project{}, err
	}
	if !exists {
		return Project{}, sql.ErrNoRows
	}

	if state.authz != nil {
		if err := state.authz.deleteProject(projectID); err != nil {
			return Project{}, err
		}
	}

	state.mu.Lock()
	delete(state.projects, projectID)
	delete(state.projectConfigs, projectID)
	for id, conv := range state.conversations {
		if conv.ProjectID != projectID {
			continue
		}
		for executionID, execution := range state.executions {
			if execution.ConversationID != id {
				continue
			}
			delete(state.executions, executionID)
			delete(state.executionDiffs, executionID)
		}
		delete(state.conversations, id)
		delete(state.conversationMessages, id)
		delete(state.conversationSnapshots, id)
		delete(state.conversationExecutionOrder, id)
		delete(state.executionEvents, id)
		delete(state.conversationEventSeq, id)
		if subscribers, ok := state.conversationEventSubs[id]; ok {
			for subID := range subscribers {
				unregisterConversationEventSubscriberLocked(state, id, subID)
			}
		}
	}
	state.mu.Unlock()
	return project, nil
}

func getProjectConfigFromStore(state *AppState, project Project) (ProjectConfig, error) {
	if strings.TrimSpace(project.ID) == "" {
		return ProjectConfig{}, errors.New("project_id is required")
	}
	if state.authz != nil {
		item, exists, err := state.authz.getProjectConfig(project.ID)
		if err != nil {
			return ProjectConfig{}, err
		}
		if exists {
			state.mu.Lock()
			state.projectConfigs[project.ID] = item
			state.mu.Unlock()
			return item, nil
		}
		return defaultProjectConfig(project.ID, project.DefaultModelID, project.UpdatedAt), nil
	}

	state.mu.RLock()
	item, exists := state.projectConfigs[project.ID]
	state.mu.RUnlock()
	if exists {
		return item, nil
	}
	return defaultProjectConfig(project.ID, project.DefaultModelID, project.UpdatedAt), nil
}

func saveProjectConfigToStore(state *AppState, workspaceID string, config ProjectConfig) (ProjectConfig, error) {
	item := config
	var err error
	if state.authz != nil {
		item, err = state.authz.upsertProjectConfig(workspaceID, config)
		if err != nil {
			return ProjectConfig{}, err
		}
	}
	state.mu.Lock()
	state.projectConfigs[item.ProjectID] = item
	state.mu.Unlock()
	return item, nil
}

func listWorkspaceProjectConfigItemsFromStore(state *AppState, workspaceID string) ([]workspaceProjectConfigItem, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return []workspaceProjectConfigItem{}, nil
	}
	if state.authz != nil {
		items, err := state.authz.listWorkspaceProjectConfigItems(workspaceID)
		if err != nil {
			return nil, err
		}
		state.mu.Lock()
		for _, item := range items {
			state.projectConfigs[item.ProjectID] = item.Config
		}
		state.mu.Unlock()
		return items, nil
	}

	state.mu.RLock()
	items := make([]workspaceProjectConfigItem, 0)
	for _, project := range state.projects {
		if project.WorkspaceID != workspaceID {
			continue
		}
		config := state.projectConfigs[project.ID]
		if strings.TrimSpace(config.ProjectID) == "" {
			config = defaultProjectConfig(project.ID, project.DefaultModelID, project.UpdatedAt)
		}
		items = append(items, workspaceProjectConfigItem{
			ProjectID:   project.ID,
			ProjectName: project.Name,
			Config:      config,
		})
	}
	state.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].ProjectName) < strings.ToLower(items[j].ProjectName)
	})
	return items, nil
}

func defaultProjectConfig(projectID string, defaultModelID string, updatedAt string) ProjectConfig {
	config := ProjectConfig{
		ProjectID: strings.TrimSpace(projectID),
		ModelIDs:  []string{},
		RuleIDs:   []string{},
		SkillIDs:  []string{},
		MCPIDs:    []string{},
		UpdatedAt: strings.TrimSpace(updatedAt),
	}
	if config.UpdatedAt == "" {
		config.UpdatedAt = nowUTC()
	}
	normalizedDefaultModelID := strings.TrimSpace(defaultModelID)
	if normalizedDefaultModelID != "" {
		config.ModelIDs = []string{normalizedDefaultModelID}
		config.DefaultModelID = toStringPtr(normalizedDefaultModelID)
	}
	return config
}
