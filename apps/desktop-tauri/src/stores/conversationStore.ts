import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export interface ConversationSessionSummary {
  session_id: string;
  project_id: string;
  title: string;
  updated_at: string;
  last_execution_id?: string;
  last_status?: string;
  last_input_preview?: string;
}

interface ConversationStoreState {
  selectedProjectId?: string;
  selectedSessionId?: string;
  sessionsByProjectId: Record<string, ConversationSessionSummary[]>;
  setSelectedProject: (projectId?: string) => void;
  setSelectedSession: (projectId: string, sessionId?: string) => void;
  setSessions: (projectId: string, sessions: ConversationSessionSummary[]) => void;
  upsertSession: (session: ConversationSessionSummary) => void;
  removeSession: (projectId: string, sessionId: string) => void;
  touchSessionExecution: (sessionId: string, patch: Partial<ConversationSessionSummary>) => void;
  reset: () => void;
}

function sortSessions(items: ConversationSessionSummary[]): ConversationSessionSummary[] {
  return [...items].sort((a, b) => b.updated_at.localeCompare(a.updated_at));
}

export const useConversationStore = create<ConversationStoreState>()(
  persist(
    (set, get) => ({
      selectedProjectId: undefined,
      selectedSessionId: undefined,
      sessionsByProjectId: {},
      setSelectedProject: (projectId) => {
        const firstSessionId = projectId ? get().sessionsByProjectId[projectId]?.[0]?.session_id : undefined;
        set({
          selectedProjectId: projectId,
          selectedSessionId: firstSessionId
        });
      },
      setSelectedSession: (projectId, sessionId) => {
        set({
          selectedProjectId: projectId,
          selectedSessionId: sessionId
        });
      },
      setSessions: (projectId, sessions) => {
        const sorted = sortSessions(sessions);
        set((state) => {
          const selectedSessionId =
            state.selectedProjectId === projectId
              ? state.selectedSessionId && sorted.some((item) => item.session_id === state.selectedSessionId)
                ? state.selectedSessionId
                : sorted[0]?.session_id
              : state.selectedSessionId;

          return {
            sessionsByProjectId: {
              ...state.sessionsByProjectId,
              [projectId]: sorted
            },
            selectedSessionId
          };
        });
      },
      upsertSession: (session) => {
        set((state) => {
          const current = state.sessionsByProjectId[session.project_id] ?? [];
          const idx = current.findIndex((item) => item.session_id === session.session_id);
          const next = [...current];
          if (idx >= 0) {
            next[idx] = { ...next[idx], ...session };
          } else {
            next.push(session);
          }

          return {
            sessionsByProjectId: {
              ...state.sessionsByProjectId,
              [session.project_id]: sortSessions(next)
            }
          };
        });
      },
      removeSession: (projectId, sessionId) => {
        set((state) => {
          const current = state.sessionsByProjectId[projectId] ?? [];
          const next = current.filter((session) => session.session_id !== sessionId);
          const selectedSessionId =
            state.selectedProjectId === projectId && state.selectedSessionId === sessionId
              ? next[0]?.session_id
              : state.selectedSessionId;

          return {
            sessionsByProjectId: {
              ...state.sessionsByProjectId,
              [projectId]: next
            },
            selectedSessionId
          };
        });
      },
      touchSessionExecution: (sessionId, patch) => {
        set((state) => {
          const sessionsByProjectId: Record<string, ConversationSessionSummary[]> = {};
          for (const [projectId, sessions] of Object.entries(state.sessionsByProjectId)) {
            const next = sessions.map((session) =>
              session.session_id === sessionId
                ? {
                    ...session,
                    ...patch,
                    updated_at: patch.updated_at ?? new Date().toISOString()
                  }
                : session
            );
            sessionsByProjectId[projectId] = sortSessions(next);
          }

          return {
            sessionsByProjectId
          };
        });
      },
      reset: () => {
        set({
          selectedProjectId: undefined,
          selectedSessionId: undefined,
          sessionsByProjectId: {}
        });
      }
    }),
    {
      name: "goyais.conversations.v2",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        selectedProjectId: state.selectedProjectId,
        selectedSessionId: state.selectedSessionId,
        sessionsByProjectId: state.sessionsByProjectId
      })
    }
  )
);
