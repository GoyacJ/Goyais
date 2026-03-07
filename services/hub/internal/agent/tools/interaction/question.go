// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package interaction handles tool-result driven user interaction normalization.
package interaction

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// QuestionOption is one normalized selectable option in a user question.
type QuestionOption struct {
	ID          string
	Label       string
	Description string
}

// PendingUserQuestion is the normalized runtime representation of a tool-driven
// user question.
type PendingUserQuestion struct {
	QuestionID          string
	Question            string
	Options             []QuestionOption
	RecommendedOptionID string
	AllowText           bool
	Required            bool
	CallID              string
	ToolName            string
}

// RequiresUserInputFromToolResult reports whether tool output requests
// user-elicitation flow.
func RequiresUserInputFromToolResult(output map[string]any) bool {
	if len(output) == 0 {
		return false
	}
	required, _ := output["requires_user_input"].(bool)
	return required
}

// NormalizePendingUserQuestion converts heterogeneous tool outputs into one
// stable PendingUserQuestion payload.
func NormalizePendingUserQuestion(output map[string]any, fallbackCallID string, fallbackToolName string) PendingUserQuestion {
	questionID := lookupString(output, "question_id")
	if questionID == "" {
		questionID = strings.TrimSpace(fallbackCallID)
	}
	if questionID == "" {
		questionID = "question_" + randomHex(6)
	}

	question := lookupString(output, "question")
	if question == "" {
		question = "Please choose one option to continue."
	}

	options := []QuestionOption{}
	if rawOptions, ok := output["options"].([]any); ok {
		for idx, item := range rawOptions {
			switch typed := item.(type) {
			case string:
				label := strings.TrimSpace(typed)
				if label == "" {
					continue
				}
				options = append(options, QuestionOption{
					ID:          fmt.Sprintf("option_%d", idx+1),
					Label:       label,
					Description: "",
				})
			case map[string]any:
				id := lookupString(typed, "id")
				label := lookupString(typed, "label")
				description := lookupString(typed, "description")
				if label == "" {
					continue
				}
				if id == "" {
					id = fmt.Sprintf("option_%d", idx+1)
				}
				options = append(options, QuestionOption{
					ID:          id,
					Label:       label,
					Description: description,
				})
			}
		}
	}

	recommendedOptionID := lookupString(output, "recommended_option_id")
	allowText, hasAllowText := output["allow_text"].(bool)
	if !hasAllowText {
		allowText = true
	}
	required, hasRequired := output["required"].(bool)
	if !hasRequired {
		required = true
	}

	return PendingUserQuestion{
		QuestionID:          questionID,
		Question:            question,
		Options:             options,
		RecommendedOptionID: recommendedOptionID,
		AllowText:           allowText,
		Required:            required,
		CallID:              strings.TrimSpace(fallbackCallID),
		ToolName:            strings.TrimSpace(fallbackToolName),
	}
}

func randomHex(bytesLen int) string {
	if bytesLen <= 0 {
		return ""
	}
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return strings.ToLower(hex.EncodeToString(buf))
}

func lookupString(source map[string]any, key string) string {
	if len(source) == 0 {
		return ""
	}
	value, exists := source[key]
	if !exists || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
