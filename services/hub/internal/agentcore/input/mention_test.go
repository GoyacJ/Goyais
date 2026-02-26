package input

import (
	"strings"
	"testing"
)

func TestProcessMentions_ValidRunAgentMention(t *testing.T) {
	result := ProcessMentions("please handle this @run-agent-reviewer now", []string{"reviewer"})
	if len(result.AgentMentions) != 1 {
		t.Fatalf("expected 1 agent mention, got %d", len(result.AgentMentions))
	}
	mention := result.AgentMentions[0]
	if mention.Kind != MentionKindRunAgent || mention.Target != "reviewer" || !mention.Known {
		t.Fatalf("unexpected mention parsing result: %+v", mention)
	}
	if !strings.Contains(result.Prompt, "run-agent:reviewer") {
		t.Fatalf("expected routing directive in prompt, got %q", result.Prompt)
	}
	if strings.Contains(result.Prompt, "@run-agent-reviewer") {
		t.Fatalf("expected mention token removed from user prompt, got %q", result.Prompt)
	}
}

func TestProcessMentions_ValidAskMention(t *testing.T) {
	result := ProcessMentions("summarize this @ask-architect", []string{"architect"})
	if len(result.AgentMentions) != 1 {
		t.Fatalf("expected 1 ask mention, got %d", len(result.AgentMentions))
	}
	if result.AgentMentions[0].Kind != MentionKindAskModel {
		t.Fatalf("expected ask mention kind, got %+v", result.AgentMentions[0])
	}
	if !strings.Contains(result.Prompt, "ask-model:architect") {
		t.Fatalf("expected ask-model directive, got %q", result.Prompt)
	}
}

func TestProcessMentions_UnknownMentionIsReported(t *testing.T) {
	result := ProcessMentions("@run-agent-ghost investigate this", []string{"reviewer"})
	if len(result.UnknownMentions) != 1 {
		t.Fatalf("expected 1 unknown mention, got %d", len(result.UnknownMentions))
	}
	if result.UnknownMentions[0].Raw != "@run-agent-ghost" {
		t.Fatalf("unexpected unknown mention: %+v", result.UnknownMentions[0])
	}
	if !strings.Contains(result.Prompt, "unknown mention ignored: @run-agent-ghost") {
		t.Fatalf("expected unknown mention warning in prompt, got %q", result.Prompt)
	}
	if !strings.Contains(result.Prompt, "investigate this") {
		t.Fatalf("expected original user text retained, got %q", result.Prompt)
	}
}

func TestProcessMentions_CoexistsWithFileMentions(t *testing.T) {
	prompt := "@run-agent-reviewer inspect @src/main.go and @\"docs/read me.md\""
	result := ProcessMentions(prompt, []string{"reviewer"})
	if !strings.Contains(result.Prompt, "@src/main.go") {
		t.Fatalf("expected @file mention to stay in prompt, got %q", result.Prompt)
	}
	if !strings.Contains(result.Prompt, "@\"docs/read me.md\"") {
		t.Fatalf("expected quoted @file mention to stay in prompt, got %q", result.Prompt)
	}
}
