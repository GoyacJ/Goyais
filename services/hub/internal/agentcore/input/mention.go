package input

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type AgentMentionKind string

const (
	MentionKindAskModel AgentMentionKind = "ask"
	MentionKindRunAgent AgentMentionKind = "run_agent"
)

type AgentMention struct {
	Raw    string
	Target string
	Kind   AgentMentionKind
	Known  bool
}

type MentionProcessingResult struct {
	Prompt          string
	AgentMentions   []AgentMention
	UnknownMentions []AgentMention
}

var (
	askMentionPattern      = regexp.MustCompile(`@ask-([\w-]+)`)
	runAgentMentionPattern = regexp.MustCompile(`@run-agent-([\w-]+)`)
)

func ProcessMentions(prompt string, knownAgents []string) MentionProcessingResult {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return MentionProcessingResult{Prompt: ""}
	}

	knownSet := map[string]struct{}{}
	for _, item := range knownAgents {
		token := normalizeMentionToken(item)
		if token == "" {
			continue
		}
		knownSet[token] = struct{}{}
	}
	checkKnown := len(knownSet) > 0

	collected := make([]AgentMention, 0, 4)
	unknown := make([]AgentMention, 0, 2)
	cleanedPrompt := trimmed

	matchAndStrip := func(pattern *regexp.Regexp, kind AgentMentionKind) {
		matches := pattern.FindAllStringSubmatch(cleanedPrompt, -1)
		if len(matches) == 0 {
			return
		}
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			raw := strings.TrimSpace(match[0])
			target := normalizeMentionToken(match[1])
			if raw == "" || target == "" {
				continue
			}
			mention := AgentMention{
				Raw:    raw,
				Target: target,
				Kind:   kind,
				Known:  true,
			}
			if checkKnown {
				_, exists := knownSet[target]
				mention.Known = exists
			}
			collected = append(collected, mention)
			if !mention.Known {
				unknown = append(unknown, mention)
			}
		}
		cleanedPrompt = strings.TrimSpace(pattern.ReplaceAllString(cleanedPrompt, ""))
	}

	matchAndStrip(runAgentMentionPattern, MentionKindRunAgent)
	matchAndStrip(askMentionPattern, MentionKindAskModel)
	cleanedPrompt = compactSpaces(cleanedPrompt)

	if len(collected) == 0 {
		return MentionProcessingResult{
			Prompt:          trimmed,
			AgentMentions:   nil,
			UnknownMentions: nil,
		}
	}

	lines := make([]string, 0, len(collected)+len(unknown)+1)
	lines = append(lines, "Agent mention routing directives:")
	for _, mention := range collected {
		if !mention.Known {
			continue
		}
		switch mention.Kind {
		case MentionKindRunAgent:
			lines = append(lines, fmt.Sprintf("- run-agent:%s", mention.Target))
		case MentionKindAskModel:
			lines = append(lines, fmt.Sprintf("- ask-model:%s", mention.Target))
		}
	}
	for _, mention := range unknown {
		lines = append(lines, fmt.Sprintf("- unknown mention ignored: %s", mention.Raw))
	}

	if cleanedPrompt == "" {
		cleanedPrompt = trimmed
	}
	processedPrompt := strings.TrimSpace(strings.Join(lines, "\n") + "\n\nUser prompt:\n" + cleanedPrompt)

	// Keep mention ordering stable but unique by raw marker.
	seen := map[string]struct{}{}
	deduped := make([]AgentMention, 0, len(collected))
	for _, mention := range collected {
		key := strings.ToLower(mention.Raw + "|" + mention.Target + "|" + string(mention.Kind))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, mention)
	}
	slices.SortStableFunc(deduped, func(a, b AgentMention) int {
		if a.Raw == b.Raw {
			return strings.Compare(a.Target, b.Target)
		}
		return strings.Compare(a.Raw, b.Raw)
	})

	return MentionProcessingResult{
		Prompt:          processedPrompt,
		AgentMentions:   deduped,
		UnknownMentions: unknown,
	}
}

func normalizeMentionToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func compactSpaces(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
