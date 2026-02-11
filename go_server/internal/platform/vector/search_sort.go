// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package vector

import (
	"math"
	"sort"
)

func sortSearchResults(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		if math.Abs(results[i].Score-results[j].Score) < 1e-12 {
			return results[i].ID < results[j].ID
		}
		return results[i].Score > results[j].Score
	})
}
