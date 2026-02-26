package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestDispatchHandlesNonSlashPrompt(t *testing.T) {
	result, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "hello world"})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if result.Handled {
		t.Fatalf("expected non-slash prompt not handled: %+v", result)
	}
}

func TestDispatchDisableSlashCommands(t *testing.T) {
	result, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/help", DisableSlashCommands: true})
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if result.Handled {
		t.Fatalf("expected slash prompt to be ignored when disabled: %+v", result)
	}
}

func TestDefaultRegistry_PrimaryNamesAndAliases(t *testing.T) {
	registry := NewDefaultRegistry()
	expectedPrimary := uniqueSorted([]string{
		"help",
		"agents",
		"bug",
		"clear",
		"compact",
		"compact-threshold",
		"config",
		"cost",
		"ctx-viz",
		"doctor",
		"init",
		"listen",
		"login",
		"logout",
		"mcp",
		"messages-debug",
		"model",
		"modelstatus",
		"onboarding",
		"output-style",
		"plugin",
		"pr-comments",
		"refresh-commands",
		"release-notes",
		"rename",
		"resume",
		"review",
		"statusline",
		"tag",
		"todos",
	})
	if !reflect.DeepEqual(registry.PrimaryNames(), expectedPrimary) {
		t.Fatalf("primary slash command names mismatch\nexpected=%v\ngot=%v", expectedPrimary, registry.PrimaryNames())
	}

	for alias, canonical := range map[string]string{
		"ms":           "modelstatus",
		"model-status": "modelstatus",
		"todo":         "todos",
	} {
		resolved, ok := registry.Get(alias)
		if !ok {
			t.Fatalf("expected alias /%s to be registered", alias)
		}
		if resolved.Name != canonical {
			t.Fatalf("expected alias /%s to map to %q, got %q", alias, canonical, resolved.Name)
		}
	}
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}
	sort.Strings(ordered)
	return ordered
}

func toString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func TestDispatchHelpAndUnknown(t *testing.T) {
	helpResult, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/help"})
	if err != nil {
		t.Fatalf("dispatch help failed: %v", err)
	}
	if !helpResult.Handled {
		t.Fatalf("expected /help handled")
	}
	if !strings.Contains(helpResult.Output, "Available slash commands") {
		t.Fatalf("unexpected help output: %q", helpResult.Output)
	}
	for _, expected := range []string{"/agents", "/init", "/model", "/mcp", "/todos"} {
		if !strings.Contains(helpResult.Output, expected) {
			t.Fatalf("help output should include %s: %q", expected, helpResult.Output)
		}
	}

	unknownResult, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/not-real"})
	if err != nil {
		t.Fatalf("dispatch unknown failed: %v", err)
	}
	if !unknownResult.Handled {
		t.Fatalf("expected unknown slash handled")
	}
	if !strings.Contains(unknownResult.Output, "Unknown slash command") {
		t.Fatalf("unexpected unknown output: %q", unknownResult.Output)
	}
}

func TestDispatchTodosLifecycle(t *testing.T) {
	workdir := t.TempDir()

	addResult, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/todos add write tests", WorkingDir: workdir})
	if err != nil {
		t.Fatalf("dispatch add failed: %v", err)
	}
	if !addResult.Handled || !strings.Contains(addResult.Output, "Added todo") {
		t.Fatalf("unexpected add output: %+v", addResult)
	}

	listResult, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/todos list", WorkingDir: workdir})
	if err != nil {
		t.Fatalf("dispatch list failed: %v", err)
	}
	if !strings.Contains(listResult.Output, "write tests") {
		t.Fatalf("unexpected list output: %q", listResult.Output)
	}

	doneResult, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/todos done 1", WorkingDir: workdir})
	if err != nil {
		t.Fatalf("dispatch done failed: %v", err)
	}
	if !strings.Contains(doneResult.Output, "Marked todo 1 as done") {
		t.Fatalf("unexpected done output: %q", doneResult.Output)
	}

	todoFile := filepath.Join(workdir, ".goyais", "slash-todos.json")
	if _, err := os.Stat(todoFile); err != nil {
		t.Fatalf("expected slash todo file to exist: %v", err)
	}
}

func TestDispatchModelAndModelStatus(t *testing.T) {
	workdir := t.TempDir()

	result, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/model gpt-5-mini", WorkingDir: workdir})
	if err != nil {
		t.Fatalf("dispatch model failed: %v", err)
	}
	if !result.Handled || !strings.Contains(result.Output, "gpt-5-mini") {
		t.Fatalf("unexpected model output: %+v", result)
	}

	status, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{Prompt: "/modelstatus", WorkingDir: workdir})
	if err != nil {
		t.Fatalf("dispatch modelstatus failed: %v", err)
	}
	if !status.Handled || !strings.Contains(status.Output, "active_model") {
		t.Fatalf("unexpected modelstatus output: %+v", status)
	}
}

func TestDispatchModelCyclePersistsAndReportsStatus(t *testing.T) {
	workdir := t.TempDir()

	cycleResult, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{
		Prompt:     "/model cycle",
		WorkingDir: workdir,
	})
	if err != nil {
		t.Fatalf("dispatch model cycle failed: %v", err)
	}
	if !cycleResult.Handled {
		t.Fatalf("expected model cycle to be handled")
	}
	if !strings.Contains(cycleResult.Output, "Model selected for this session:") {
		t.Fatalf("unexpected model cycle output: %q", cycleResult.Output)
	}

	status, err := Dispatch(context.Background(), NewDefaultRegistry(), DispatchRequest{
		Prompt:     "/modelstatus",
		WorkingDir: workdir,
	})
	if err != nil {
		t.Fatalf("dispatch modelstatus failed: %v", err)
	}
	if !strings.Contains(status.Output, "active_model:") {
		t.Fatalf("expected active model in status output, got %q", status.Output)
	}

	statePath := filepath.Join(workdir, ".goyais", "slash-state.json")
	rawState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read slash state: %v", err)
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(rawState, &decoded); err != nil {
		t.Fatalf("parse slash state: %v", err)
	}
	selectedModel := strings.TrimSpace(toString(decoded["model"]))
	if selectedModel == "" {
		t.Fatalf("expected persisted model in slash state, got %q", string(rawState))
	}
	if !strings.Contains(status.Output, selectedModel) {
		t.Fatalf("expected status output to include persisted model %q, got %q", selectedModel, status.Output)
	}
}

func TestDispatchAllCoreCommandsReachable(t *testing.T) {
	workdir := t.TempDir()
	registry := NewDefaultRegistry()

	testPrompts := []string{
		"/agents",
		"/bug test",
		"/clear",
		"/compact",
		"/compact-threshold 2000",
		"/config",
		"/cost",
		"/ctx-viz",
		"/doctor",
		"/help",
		"/init",
		"/listen on",
		"/login",
		"/logout",
		"/mcp",
		"/messages-debug on",
		"/model",
		"/modelstatus",
		"/ms",
		"/model-status",
		"/onboarding",
		"/output-style compact",
		"/plugin",
		"/pr-comments looks good",
		"/refresh-commands",
		"/release-notes",
		"/rename session-a",
		"/resume session-a",
		"/review services/hub",
		"/statusline on",
		"/tag release",
		"/todos list",
		"/todo list",
	}

	for _, prompt := range testPrompts {
		prompt := prompt
		t.Run(prompt, func(t *testing.T) {
			result, err := Dispatch(context.Background(), registry, DispatchRequest{Prompt: prompt, WorkingDir: workdir, Env: map[string]string{"GOYAIS_MODEL": "gpt-5"}})
			if err != nil {
				t.Fatalf("dispatch %s failed: %v", prompt, err)
			}
			if !result.Handled {
				t.Fatalf("expected prompt %q to be handled", prompt)
			}
			if strings.Contains(result.Output, "Unknown slash command") {
				t.Fatalf("expected known command for %q, got output %q", prompt, result.Output)
			}
		})
	}
}
