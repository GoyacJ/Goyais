// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

// DiffItem captures one normalized file-level change artifact produced by tools.
type DiffItem struct {
	ID           string
	Path         string
	ChangeType   string
	Summary      string
	AddedLines   *int
	DeletedLines *int
	BeforeBlob   string
	AfterBlob    string
}
