package composer

import "testing"

func TestSuggestCommand(t *testing.T) {
	suggestions := Suggest(SuggestRequest{
		Draft:  "/he",
		Cursor: 3,
		Commands: []CommandMeta{
			{Name: "help", Description: "Show available commands", Kind: CommandKindControl},
			{Name: "model", Description: "model", Kind: CommandKindControl},
		},
	})
	if len(suggestions) == 0 {
		t.Fatalf("expected command suggestions")
	}
	if suggestions[0].InsertText != "/help" {
		t.Fatalf("unexpected suggestion: %+v", suggestions[0])
	}
	if suggestions[0].Detail != "Show available commands" {
		t.Fatalf("expected command detail, got %q", suggestions[0].Detail)
	}
}

func TestSuggestResource(t *testing.T) {
	suggestions := Suggest(SuggestRequest{
		Draft:  "@rule:r",
		Cursor: 7,
		Resources: []ResourceCatalogItem{
			{Type: ResourceTypeRule, ID: "rule_alpha", Name: "Rule Alpha Display"},
			{Type: ResourceTypeSkill, ID: "skill_alpha", Name: "Skill Alpha"},
		},
	})
	if len(suggestions) != 1 {
		t.Fatalf("expected one resource suggestion, got %+v", suggestions)
	}
	if suggestions[0].InsertText != "@rule:rule_alpha" {
		t.Fatalf("unexpected insert text: %q", suggestions[0].InsertText)
	}
	if suggestions[0].Detail != "Rule Alpha Display" {
		t.Fatalf("expected resource detail, got %q", suggestions[0].Detail)
	}
}

func TestSuggestFileResourceByPathAndBasename(t *testing.T) {
	suggestions := Suggest(SuggestRequest{
		Draft:  "@file:main",
		Cursor: 10,
		Resources: []ResourceCatalogItem{
			{Type: ResourceTypeFile, ID: "src/main.ts", Name: "main.ts"},
			{Type: ResourceTypeFile, ID: "README.md", Name: "README.md"},
		},
	})
	if len(suggestions) != 1 {
		t.Fatalf("expected one file suggestion, got %+v", suggestions)
	}
	if suggestions[0].InsertText != "@file:src/main.ts" {
		t.Fatalf("unexpected insert text: %q", suggestions[0].InsertText)
	}
	if suggestions[0].Detail != "" {
		t.Fatalf("expected empty file detail, got %q", suggestions[0].Detail)
	}
}
