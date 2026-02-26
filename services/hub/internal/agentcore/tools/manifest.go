package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type ManifestEntry struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	SchemaHash       string `json:"schema_hash"`
	RiskLevel        string `json:"risk_level"`
	ReadOnly         bool   `json:"read_only"`
	ConcurrencySafe  bool   `json:"concurrency_safe"`
	NeedsPermissions bool   `json:"needs_permissions"`
}

func BuildManifest(registry *Registry) []ManifestEntry {
	if registry == nil {
		return nil
	}
	tools := registry.ListOrdered()
	out := make([]ManifestEntry, 0, len(tools))
	for _, tool := range tools {
		spec := tool.Spec()
		out = append(out, ManifestEntry{
			Name:             spec.Name,
			Description:      spec.Description,
			SchemaHash:       hashSchema(spec.InputSchema),
			RiskLevel:        string(spec.RiskLevel),
			ReadOnly:         spec.ReadOnly,
			ConcurrencySafe:  spec.ConcurrencySafe,
			NeedsPermissions: spec.NeedsPermissions,
		})
	}
	return out
}

func hashSchema(schema map[string]any) string {
	raw, err := json.Marshal(schema)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}
