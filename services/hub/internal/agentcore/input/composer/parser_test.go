package composer

import "testing"

func TestParsePromptAndMentions(t *testing.T) {
	result := Parse("请处理 @rule:rule_a 和 @skill:skill_a")
	if result.IsCommand {
		t.Fatalf("expected non-command parse")
	}
	if got := result.PromptText; got != "请处理 和" {
		t.Fatalf("unexpected prompt text: %q", got)
	}
	if len(result.MentionedRefs) != 2 {
		t.Fatalf("expected 2 mentions, got %d", len(result.MentionedRefs))
	}
}

func TestParseCommand(t *testing.T) {
	result := Parse("/help")
	if !result.IsCommand {
		t.Fatalf("expected command parse")
	}
	if result.CommandText != "/help" {
		t.Fatalf("unexpected command text: %q", result.CommandText)
	}
}

func TestParsePromptInjectsFilePathMention(t *testing.T) {
	result := Parse("请查看 @file:src/main.ts 并遵循 @rule:rule_a")
	if result.IsCommand {
		t.Fatalf("expected non-command parse")
	}
	if got := result.PromptText; got != "请查看 src/main.ts 并遵循" {
		t.Fatalf("unexpected prompt text: %q", got)
	}
	if len(result.MentionedRefs) != 2 {
		t.Fatalf("expected 2 mentions, got %d", len(result.MentionedRefs))
	}
	if result.MentionedRefs[0].Type != ResourceTypeFile || result.MentionedRefs[0].ID != "src/main.ts" {
		t.Fatalf("expected file mention first, got %+v", result.MentionedRefs[0])
	}
}
