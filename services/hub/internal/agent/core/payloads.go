// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import eventscore "goyais/services/hub/internal/agent/core/events"

// RunQueuedPayload captures metadata when a run enters the session queue.
type RunQueuedPayload = eventscore.RunQueuedPayload

// RunStartedPayload marks that a run has begun active execution.
type RunStartedPayload = eventscore.RunStartedPayload

// OutputDeltaPayload carries incremental model output chunks.
type OutputDeltaPayload = eventscore.OutputDeltaPayload

// ApprovalNeededPayload describes a permission checkpoint before tool use.
type ApprovalNeededPayload = eventscore.ApprovalNeededPayload

// RunFailedPayload describes a terminal failure with structured metadata.
type RunFailedPayload = eventscore.RunFailedPayload

// RunCompletedPayload summarizes completion metadata for a successful run.
type RunCompletedPayload = eventscore.RunCompletedPayload

// RunCancelledPayload captures who/what cancelled the run.
type RunCancelledPayload = eventscore.RunCancelledPayload
