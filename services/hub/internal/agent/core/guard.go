// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

// Compile-time guards for payload-to-contract conformance.
var (
	_ EventPayload = OutputDeltaPayload{}
	_ EventPayload = ApprovalNeededPayload{}
	_ EventPayload = RunFailedPayload{}
	_ EventPayload = RunCompletedPayload{}
	_ EventPayload = RunCancelledPayload{}
)
