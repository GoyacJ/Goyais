package input

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompletionSuggest_NoAtAutoPrefixForAgentAndModel(t *testing.T) {
	engine := NewCompletionEngine()
	agentSuggestions := engine.Suggest(CompletionRequest{
		Input:        "run-agent-rev",
		AgentTargets: []string{"reviewer"},
		ModelTargets: []string{"architect"},
		MaxResults:   20,
	})
	if len(agentSuggestions) == 0 {
		t.Fatal("expected non-empty agent suggestions")
	}
	hasRunAgent := false
	for _, suggestion := range agentSuggestions {
		if suggestion.InsertText == "@run-agent-reviewer" {
			hasRunAgent = true
		}
	}
	if !hasRunAgent {
		t.Fatalf("expected @run-agent-reviewer suggestion, got %+v", agentSuggestions)
	}

	modelSuggestions := engine.Suggest(CompletionRequest{
		Input:        "ask-arch",
		AgentTargets: []string{"reviewer"},
		ModelTargets: []string{"architect"},
		MaxResults:   20,
	})
	hasAsk := false
	for _, suggestion := range modelSuggestions {
		if suggestion.InsertText == "@ask-architect" {
			hasAsk = true
		}
	}
	if !hasAsk {
		t.Fatalf("expected @ask-architect suggestion, got %+v", modelSuggestions)
	}
}

func TestCompletionSuggest_StableOrderingForEqualScores(t *testing.T) {
	engine := NewCompletionEngine()
	suggestions := engine.Suggest(CompletionRequest{
		Input:         "/m",
		SlashCommands: []string{"model", "mcp", "merge"},
		MaxResults:    10,
	})
	if len(suggestions) < 3 {
		t.Fatalf("expected at least 3 suggestions, got %d", len(suggestions))
	}
	if suggestions[0].InsertText != "/mcp" && suggestions[0].InsertText != "/model" && suggestions[0].InsertText != "/merge" {
		t.Fatalf("unexpected first suggestion: %+v", suggestions[0])
	}
	// Deterministic ordering on repeated calls.
	again := engine.Suggest(CompletionRequest{
		Input:         "/m",
		SlashCommands: []string{"model", "mcp", "merge"},
		MaxResults:    10,
	})
	for idx := range suggestions {
		if idx >= len(again) {
			t.Fatalf("unexpected length mismatch")
		}
		if suggestions[idx].InsertText != again[idx].InsertText {
			t.Fatalf("expected stable ordering, got %q then %q at index %d", suggestions[idx].InsertText, again[idx].InsertText, idx)
		}
	}
}

func TestCompletionSuggest_UnixCommandIntersectionAndPriority(t *testing.T) {
	if !isUnixLikePlatform(runtime.GOOS) {
		t.Skip("unix command completion only applies to linux/darwin")
	}
	tmp := t.TempDir()
	mustWriteExecutable(t, filepath.Join(tmp, "grep"))
	mustWriteExecutable(t, filepath.Join(tmp, "gizmo"))
	mustWriteExecutable(t, filepath.Join(tmp, "git"))

	engine := NewCompletionEngine()
	suggestions := engine.Suggest(CompletionRequest{
		Input:      "g",
		MaxResults: 10,
		Env: map[string]string{
			"PATH": tmp,
		},
	})
	if len(suggestions) == 0 {
		t.Fatal("expected command suggestions from PATH")
	}
	first := suggestions[0]
	if first.Kind != CompletionKindCommand {
		t.Fatalf("expected command suggestion first, got %+v", first)
	}
	if first.InsertText != "git" && first.InsertText != "grep" {
		t.Fatalf("expected common command priority, got %q", first.InsertText)
	}
}

func TestCompletionSuggest_PathCommandDiscoveryUsesCache(t *testing.T) {
	if !isUnixLikePlatform(runtime.GOOS) {
		t.Skip("unix command completion only applies to linux/darwin")
	}
	tmp := t.TempDir()
	commandPath := filepath.Join(tmp, "cachedcmd")
	mustWriteExecutable(t, commandPath)

	engine := NewCompletionEngine()
	first := engine.Suggest(CompletionRequest{
		Input:      "cached",
		MaxResults: 10,
		Env: map[string]string{
			"PATH": tmp,
		},
	})
	if len(first) == 0 {
		t.Fatal("expected first PATH discovery to return cachedcmd")
	}

	if err := os.Remove(commandPath); err != nil {
		t.Fatalf("remove command path: %v", err)
	}

	second := engine.Suggest(CompletionRequest{
		Input:      "cached",
		MaxResults: 10,
		Env: map[string]string{
			"PATH": tmp,
		},
	})
	hasCached := false
	for _, suggestion := range second {
		if suggestion.InsertText == "cachedcmd" {
			hasCached = true
			break
		}
	}
	if !hasCached {
		t.Fatalf("expected cached discovery result to contain cachedcmd, got %+v", second)
	}
}

func TestCompletionSuggest_FileAndSlashDedup(t *testing.T) {
	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	engine := NewCompletionEngine()
	suggestions := engine.Suggest(CompletionRequest{
		Input:         "@ma",
		WorkingDir:    workdir,
		SlashCommands: []string{"main"},
		AgentTargets:  []string{"main"},
		MaxResults:    20,
	})
	seen := map[string]struct{}{}
	for _, suggestion := range suggestions {
		if _, exists := seen[suggestion.InsertText]; exists {
			t.Fatalf("expected deduped insert texts, got duplicate %q in %+v", suggestion.InsertText, suggestions)
		}
		seen[suggestion.InsertText] = struct{}{}
	}
}

func TestIsUnixLikePlatformCoverage(t *testing.T) {
	if !isUnixLikePlatform("linux") {
		t.Fatal("linux should be treated as unix-like")
	}
	if !isUnixLikePlatform("darwin") {
		t.Fatal("darwin should be treated as unix-like")
	}
	if isUnixLikePlatform("windows") {
		t.Fatal("windows should not be treated as unix-like")
	}
}

func mustWriteExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}
