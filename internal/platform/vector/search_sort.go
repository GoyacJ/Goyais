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
