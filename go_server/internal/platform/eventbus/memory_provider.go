// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package eventbus

import "context"

type MemoryProvider struct{}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{}
}

func (p *MemoryProvider) Publish(context.Context, string, Message) error {
	return nil
}

func (p *MemoryProvider) Ping(context.Context) error {
	return nil
}

func (p *MemoryProvider) Close() error {
	return nil
}

func (p *MemoryProvider) Name() string {
	return "memory"
}
