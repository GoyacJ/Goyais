// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package rulesdsl

import "testing"

func TestParseRule(t *testing.T) {
	rule, err := ParseRule(`deny Bash(npm run *)`)
	if err != nil {
		t.Fatalf("parse rule failed: %v", err)
	}
	if rule.Effect != EffectDeny {
		t.Fatalf("unexpected effect %q", rule.Effect)
	}
	if rule.Tool != "Bash" {
		t.Fatalf("unexpected tool %q", rule.Tool)
	}
	if rule.Pattern != "npm run *" {
		t.Fatalf("unexpected pattern %q", rule.Pattern)
	}
}

func TestMatchReadRelativeAndAbsoluteDoubleSlash(t *testing.T) {
	relative, err := ParseRule(`allow Read(./.env)`)
	if err != nil {
		t.Fatalf("parse relative rule failed: %v", err)
	}
	if !Match(relative, Request{Tool: "Read", Argument: "./.env"}) {
		t.Fatal("expected relative read rule to match")
	}

	absolute, err := ParseRule(`deny Read(//etc/*)`)
	if err != nil {
		t.Fatalf("parse absolute rule failed: %v", err)
	}
	if !Match(absolute, Request{Tool: "Read", Argument: "/etc/passwd"}) {
		t.Fatal("expected // absolute read rule to match /etc/passwd")
	}
}

func TestBashShellOperatorAwareness(t *testing.T) {
	rule, err := ParseRule(`allow Bash(npm run *)`)
	if err != nil {
		t.Fatalf("parse rule failed: %v", err)
	}
	if !Match(rule, Request{Tool: "Bash", Argument: "npm run lint"}) {
		t.Fatal("expected simple npm run to match")
	}
	if Match(rule, Request{Tool: "Bash", Argument: "npm run lint && rm -rf /"}) {
		t.Fatal("operator-bearing bash command should not match operator-free pattern")
	}
}

func TestEvaluatePrecedence(t *testing.T) {
	rules, err := ParseLines([]string{
		`allow Read(./*)`,
		`ask Read(./secret.txt)`,
		`deny Read(./secret.txt)`,
	})
	if err != nil {
		t.Fatalf("parse lines failed: %v", err)
	}
	effect, matched := Evaluate(rules, Request{
		Tool:     "Read",
		Argument: "./secret.txt",
	})
	if effect != EffectDeny {
		t.Fatalf("expected deny precedence, got %q (matched=%#v)", effect, matched)
	}
	if len(matched) != 3 {
		t.Fatalf("expected 3 matched rules, got %#v", matched)
	}
}
