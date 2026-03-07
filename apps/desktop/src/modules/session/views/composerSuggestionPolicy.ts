import type { ComposerCatalog, ComposerSuggestion } from "@/shared/types/api";

export type SuggestionContextKind = "slash" | "resource" | "none";

export type SuggestionContext = {
  kind: SuggestionContextKind;
  query: string;
  tokenRange: {
    start: number;
    end: number;
  };
  isFileToken: boolean;
};

export function resolveSuggestionContext(draft: string, cursor: number): SuggestionContext {
  const safeCursor = Math.max(0, Math.min(cursor, draft.length));
  const { tokenStart, tokenEnd, token } = resolveActiveToken(draft, safeCursor);
  const trimmedToken = token.trim();
  if (trimmedToken.startsWith("/")) {
    return {
      kind: "slash",
      query: trimmedToken.slice(1),
      tokenRange: {
        start: tokenStart,
        end: tokenEnd
      },
      isFileToken: false
    };
  }
  if (trimmedToken.startsWith("@")) {
    const query = trimmedToken.slice(1);
    return {
      kind: "resource",
      query,
      tokenRange: {
        start: tokenStart,
        end: tokenEnd
      },
      isFileToken: query.toLowerCase().startsWith("file:")
    };
  }
  return {
    kind: "none",
    query: "",
    tokenRange: {
      start: tokenStart,
      end: tokenEnd
    },
    isFileToken: false
  };
}

export function resolveLocalSuggestionLimit(
  context: SuggestionContext,
  catalog: ComposerCatalog,
  resourceLimit = 12
): number {
  if (context.kind === "slash") {
    return catalog.commands.length;
  }
  if (context.kind === "resource") {
    return resourceLimit;
  }
  return 0;
}

export function shouldRequestRemoteSuggestions(context: SuggestionContext, localCount: number): boolean {
  if (context.kind === "slash") {
    return false;
  }
  if (context.kind === "resource" && context.isFileToken) {
    return true;
  }
  return localCount > 0;
}

export function dedupeComposerSuggestions(suggestions: ComposerSuggestion[]): ComposerSuggestion[] {
  if (suggestions.length <= 1) {
    return suggestions;
  }
  const seen = new Set<string>();
  const out: ComposerSuggestion[] = [];
  for (const suggestion of suggestions) {
    const key = `${suggestion.kind}|${suggestion.insert_text}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    out.push(suggestion);
  }
  return out;
}

function resolveActiveToken(draft: string, cursor: number): { tokenStart: number; tokenEnd: number; token: string } {
  let tokenStart = cursor;
  while (tokenStart > 0 && !/\s/.test(draft[tokenStart - 1] ?? "")) {
    tokenStart -= 1;
  }

  let tokenEnd = cursor;
  while (tokenEnd < draft.length && !/\s/.test(draft[tokenEnd] ?? "")) {
    tokenEnd += 1;
  }

  return {
    tokenStart,
    tokenEnd,
    token: draft.slice(tokenStart, cursor)
  };
}
