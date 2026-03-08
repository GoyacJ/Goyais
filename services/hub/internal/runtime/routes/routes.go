package routes

import "net/http"

type Handlers struct {
	Projects                       http.HandlerFunc
	ProjectsImport                 http.HandlerFunc
	ProjectByID                    http.HandlerFunc
	ProjectConversations           http.HandlerFunc
	ProjectConfig                  http.HandlerFunc
	ProjectFiles                   http.HandlerFunc
	ProjectFileContent             http.HandlerFunc
	Conversations                  http.HandlerFunc
	ConversationByID               http.HandlerFunc
	ConversationInputCatalog       http.HandlerFunc
	ConversationInputSuggest       http.HandlerFunc
	ConversationInputSubmit        http.HandlerFunc
	ConversationEvents             http.HandlerFunc
	ConversationStop               http.HandlerFunc
	ConversationExport             http.HandlerFunc
	ConversationChangeSet          http.HandlerFunc
	ConversationChangeSetCommit    http.HandlerFunc
	ConversationChangeSetDiscard   http.HandlerFunc
	ConversationChangeSetExport    http.HandlerFunc
	ConversationCheckpoints        http.HandlerFunc
	ConversationCheckpointRollback http.HandlerFunc
	Executions                     http.HandlerFunc
	RunControl                     http.HandlerFunc
	RunGraph                       http.HandlerFunc
	RunTasks                       http.HandlerFunc
	RunTaskByID                    http.HandlerFunc
	RunTaskControl                 http.HandlerFunc
}

func Register(mux *http.ServeMux, handlers Handlers) {
	mustHandle(mux, "/v1/projects", handlers.Projects)
	mustHandle(mux, "/v1/projects/import", handlers.ProjectsImport)
	mustHandle(mux, "/v1/projects/{project_id}", handlers.ProjectByID)
	mustHandle(mux, "/v1/projects/{project_id}/sessions", handlers.ProjectConversations)
	mustHandle(mux, "/v1/projects/{project_id}/config", handlers.ProjectConfig)
	mustHandle(mux, "/v1/projects/{project_id}/files", handlers.ProjectFiles)
	mustHandle(mux, "/v1/projects/{project_id}/files/content", handlers.ProjectFileContent)
	mustHandle(mux, "/v1/sessions", handlers.Conversations)
	mustHandle(mux, "/v1/sessions/{session_id}", handlers.ConversationByID)
	mustHandle(mux, "/v1/sessions/{session_id}/input/catalog", handlers.ConversationInputCatalog)
	mustHandle(mux, "/v1/sessions/{session_id}/input/suggest", handlers.ConversationInputSuggest)
	mustHandle(mux, "/v1/sessions/{session_id}/runs", handlers.ConversationInputSubmit)
	mustHandle(mux, "/v1/sessions/{session_id}/events", handlers.ConversationEvents)
	mustHandle(mux, "/v1/sessions/{session_id}/stop", handlers.ConversationStop)
	mustHandle(mux, "/v1/sessions/{session_id}/export", handlers.ConversationExport)
	mustHandle(mux, "/v1/sessions/{session_id}/changeset", handlers.ConversationChangeSet)
	mustHandle(mux, "/v1/sessions/{session_id}/changeset/commit", handlers.ConversationChangeSetCommit)
	mustHandle(mux, "/v1/sessions/{session_id}/changeset/discard", handlers.ConversationChangeSetDiscard)
	mustHandle(mux, "/v1/sessions/{session_id}/changeset/export", handlers.ConversationChangeSetExport)
	mustHandle(mux, "/v1/sessions/{session_id}/checkpoints", handlers.ConversationCheckpoints)
	mustHandle(mux, "/v1/sessions/{session_id}/checkpoints/{checkpoint_id}/rollback", handlers.ConversationCheckpointRollback)
	mustHandle(mux, "/v1/runs", handlers.Executions)
	mustHandle(mux, "/v1/runs/{run_id}/control", handlers.RunControl)
	mustHandle(mux, "/v1/runs/{run_id}/graph", handlers.RunGraph)
	mustHandle(mux, "/v1/runs/{run_id}/tasks", handlers.RunTasks)
	mustHandle(mux, "/v1/runs/{run_id}/tasks/{task_id}", handlers.RunTaskByID)
	mustHandle(mux, "/v1/runs/{run_id}/tasks/{task_id}/control", handlers.RunTaskControl)
}

func mustHandle(mux *http.ServeMux, pattern string, handler http.HandlerFunc) {
	if handler == nil {
		panic("runtime routes: nil handler for " + pattern)
	}
	mux.HandleFunc(pattern, handler)
}
