import type { SyncDatabase } from "./db";

export interface SyncMetrics {
  push_requests_total: number;
  pull_requests_total: number;
  auth_fail_total: number;
}

export function createMetrics(): SyncMetrics {
  return {
    push_requests_total: 0,
    pull_requests_total: 0,
    auth_fail_total: 0
  };
}

export function metricsSnapshot(db: SyncDatabase, metrics: SyncMetrics): Record<string, number> {
  return {
    events_total: db.countEvents(),
    push_requests_total: metrics.push_requests_total,
    pull_requests_total: metrics.pull_requests_total,
    auth_fail_total: metrics.auth_fail_total
  };
}
