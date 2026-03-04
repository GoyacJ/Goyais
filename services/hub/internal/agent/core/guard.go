// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

// Compile-time guards for payload-to-contract conformance.
var (
	_ EventPayload = RunQueuedPayload{}
	_ EventPayload = RunStartedPayload{}
	_ EventPayload = OutputDeltaPayload{}
	_ EventPayload = ApprovalNeededPayload{}
	_ EventPayload = RunFailedPayload{}
	_ EventPayload = RunCompletedPayload{}
	_ EventPayload = RunCancelledPayload{}
)

// Compile-time guards for event-type to payload bindings.
var (
	_ EventSpec[RunQueuedPayload]      = RunQueuedEventSpec
	_ EventSpec[RunStartedPayload]     = RunStartedEventSpec
	_ EventSpec[OutputDeltaPayload]    = RunOutputDeltaEventSpec
	_ EventSpec[ApprovalNeededPayload] = RunApprovalNeededEventSpec
	_ EventSpec[RunCompletedPayload]   = RunCompletedEventSpec
	_ EventSpec[RunFailedPayload]      = RunFailedEventSpec
	_ EventSpec[RunCancelledPayload]   = RunCancelledEventSpec
)
