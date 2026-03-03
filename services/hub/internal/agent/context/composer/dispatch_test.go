// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package composer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestListCommands_SortedWithKind(t *testing.T) {
	registry := NewStaticRegistry([]Command{
		{Name: "model", Description: "change model", Handler: func(context.Context, DispatchRequest, []string) (string, error) {
			return "ok", nil
		}},
		{Name: "review", Description: "run review", PromptResolver: func(context.Context, DispatchRequest, []string) ([]string, error) {
			return []string{"review prompt"}, nil
		}},
	})

	items := ListCommands(registry)
	if len(items) != 2 {
		t.Fatalf("expected 2 command metas, got %d", len(items))
	}
	if items[0].Name != "model" || items[0].Kind != CommandKindControl {
		t.Fatalf("unexpected first item %+v", items[0])
	}
	if items[1].Name != "review" || items[1].Kind != CommandKindPrompt {
		t.Fatalf("unexpected second item %+v", items[1])
	}
}

func TestDispatchCommand_Control(t *testing.T) {
	registry := NewStaticRegistry([]Command{
		{Name: "status", Handler: func(_ context.Context, req DispatchRequest, args []string) (string, error) {
			if req.WorkingDir != "/tmp/work" {
				t.Fatalf("unexpected working dir %q", req.WorkingDir)
			}
			if len(args) != 1 || args[0] != "--json" {
				t.Fatalf("unexpected args %#v", args)
			}
			return "all good", nil
		}},
	})

	result, err := DispatchCommand(context.Background(), "/status --json", registry, DispatchRequest{
		WorkingDir: "/tmp/work",
		Env: map[string]string{
			"K": "V",
		},
	})
	if err != nil {
		t.Fatalf("dispatch command: %v", err)
	}
	if result.Kind != CommandKindControl || result.Name != "status" || result.Output != "all good" {
		t.Fatalf("unexpected dispatch result %+v", result)
	}
}

func TestDispatchCommand_PromptResolver(t *testing.T) {
	registry := NewStaticRegistry([]Command{
		{Name: "review", PromptResolver: func(_ context.Context, _ DispatchRequest, args []string) ([]string, error) {
			if len(args) != 1 || args[0] != "src/app.ts" {
				t.Fatalf("unexpected args %#v", args)
			}
			return []string{"first", "second"}, nil
		}},
	})

	result, err := DispatchCommand(context.Background(), "/review src/app.ts", registry, DispatchRequest{})
	if err != nil {
		t.Fatalf("dispatch prompt command: %v", err)
	}
	if result.Kind != CommandKindPrompt {
		t.Fatalf("unexpected kind %q", result.Kind)
	}
	if result.ExpandedPrompt != "first\n\nsecond" {
		t.Fatalf("unexpected expanded prompt %q", result.ExpandedPrompt)
	}
}

func TestDispatchCommand_Unknown(t *testing.T) {
	registry := NewStaticRegistry([]Command{
		{Name: "help", Handler: func(context.Context, DispatchRequest, []string) (string, error) {
			return "ok", nil
		}},
	})
	_, err := DispatchCommand(context.Background(), "/missing", registry, DispatchRequest{})
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
}

func TestDispatchCommand_PromptResolverEmpty(t *testing.T) {
	registry := NewStaticRegistry([]Command{
		{Name: "plan", PromptResolver: func(context.Context, DispatchRequest, []string) ([]string, error) {
			return []string{"", "   "}, nil
		}},
	})
	_, err := DispatchCommand(context.Background(), "/plan", registry, DispatchRequest{})
	if err == nil {
		t.Fatal("expected empty prompt resolver error")
	}
	if !strings.Contains(err.Error(), "expanded to empty prompt") {
		t.Fatalf("unexpected error: %v", err)
	}
}
