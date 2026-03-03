// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package interaction

import (
	"regexp"
	"testing"
)

func TestRequiresUserInputFromToolResult(t *testing.T) {
	if RequiresUserInputFromToolResult(nil) {
		t.Fatal("nil output should not require user input")
	}
	if RequiresUserInputFromToolResult(map[string]any{}) {
		t.Fatal("empty output should not require user input")
	}
	if RequiresUserInputFromToolResult(map[string]any{"requires_user_input": false}) {
		t.Fatal("false requires_user_input should be false")
	}
	if !RequiresUserInputFromToolResult(map[string]any{"requires_user_input": true}) {
		t.Fatal("true requires_user_input should be true")
	}
}

func TestNormalizePendingUserQuestion_Defaults(t *testing.T) {
	question := NormalizePendingUserQuestion(map[string]any{}, "", "")

	if question.Question != "Please choose one option to continue." {
		t.Fatalf("unexpected default question: %q", question.Question)
	}
	if !regexp.MustCompile(`^question_[0-9a-f]+$`).MatchString(question.QuestionID) {
		t.Fatalf("unexpected generated question id: %q", question.QuestionID)
	}
	if !question.AllowText {
		t.Fatal("default allow_text should be true")
	}
	if !question.Required {
		t.Fatal("default required should be true")
	}
}

func TestNormalizePendingUserQuestion_UsesFallbackCallID(t *testing.T) {
	question := NormalizePendingUserQuestion(map[string]any{}, "call_123", "ask_user")
	if question.QuestionID != "call_123" {
		t.Fatalf("expected question id from fallback call, got %q", question.QuestionID)
	}
	if question.CallID != "call_123" {
		t.Fatalf("expected call id to keep fallback, got %q", question.CallID)
	}
	if question.ToolName != "ask_user" {
		t.Fatalf("expected tool name fallback, got %q", question.ToolName)
	}
}

func TestNormalizePendingUserQuestion_ParsesOptions(t *testing.T) {
	question := NormalizePendingUserQuestion(map[string]any{
		"question_id":           "q_1",
		"question":              "Choose one",
		"recommended_option_id": "opt_2",
		"allow_text":            false,
		"required":              false,
		"options": []any{
			"Option A",
			map[string]any{
				"id":          "opt_2",
				"label":       "Option B",
				"description": "Recommended",
			},
			map[string]any{
				"label": "Option C",
			},
			map[string]any{
				"id": "invalid_without_label",
			},
		},
	}, "fallback_call", "tool_name")

	if question.QuestionID != "q_1" {
		t.Fatalf("unexpected question id %q", question.QuestionID)
	}
	if question.Question != "Choose one" {
		t.Fatalf("unexpected question %q", question.Question)
	}
	if question.RecommendedOptionID != "opt_2" {
		t.Fatalf("unexpected recommended option %q", question.RecommendedOptionID)
	}
	if question.AllowText {
		t.Fatal("allow_text should be false")
	}
	if question.Required {
		t.Fatal("required should be false")
	}
	if len(question.Options) != 3 {
		t.Fatalf("expected 3 valid options, got %#v", question.Options)
	}
	if question.Options[0].ID != "option_1" || question.Options[0].Label != "Option A" {
		t.Fatalf("unexpected first option %#v", question.Options[0])
	}
	if question.Options[1].ID != "opt_2" || question.Options[1].Description != "Recommended" {
		t.Fatalf("unexpected second option %#v", question.Options[1])
	}
	if question.Options[2].ID != "option_3" || question.Options[2].Label != "Option C" {
		t.Fatalf("unexpected third option %#v", question.Options[2])
	}
}
