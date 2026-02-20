import { create } from "zustand";

interface PermissionDecision {
  runId: string;
  callId: string;
  approved: boolean;
  decidedAt: string;
}

interface PermissionState {
  decisions: PermissionDecision[];
  addDecision: (decision: PermissionDecision) => void;
}

export const usePermissionStore = create<PermissionState>((set, get) => ({
  decisions: [],
  addDecision: (decision) => set({ decisions: [...get().decisions, decision] })
}));
