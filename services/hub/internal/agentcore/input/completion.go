package input

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type CompletionKind string

const (
	CompletionKindSlash   CompletionKind = "slash"
	CompletionKindFile    CompletionKind = "file"
	CompletionKindAgent   CompletionKind = "agent"
	CompletionKindModel   CompletionKind = "model"
	CompletionKindCommand CompletionKind = "command"
)

type CompletionSuggestion struct {
	Kind       CompletionKind
	Label      string
	InsertText string
	Score      int
}

type CompletionRequest struct {
	Input         string
	WorkingDir    string
	SlashCommands []string
	AgentTargets  []string
	ModelTargets  []string
	Env           map[string]string
	MaxResults    int
}

type CompletionEngine struct {
	mu           sync.Mutex
	commandCache map[string]pathCommandCacheEntry
	cacheTTL     time.Duration
}

type pathCommandCacheEntry struct {
	commands []string
	loadedAt time.Time
}

func NewCompletionEngine() *CompletionEngine {
	return &CompletionEngine{
		commandCache: map[string]pathCommandCacheEntry{},
		cacheTTL:     30 * time.Second,
	}
}

func (e *CompletionEngine) Suggest(req CompletionRequest) []CompletionSuggestion {
	if e == nil {
		e = NewCompletionEngine()
	}
	token := lastToken(req.Input)
	if strings.TrimSpace(token) == "" {
		return nil
	}
	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}

	candidates := make([]CompletionSuggestion, 0, 64)
	candidates = append(candidates, e.suggestAgentAndModelMentions(token, req)...)
	candidates = append(candidates, e.suggestSlashCommands(token, req)...)
	candidates = append(candidates, e.suggestFiles(token, req)...)
	candidates = append(candidates, e.suggestUnixCommands(token, req)...)

	seen := map[string]struct{}{}
	unique := make([]CompletionSuggestion, 0, len(candidates))
	for _, candidate := range candidates {
		key := strings.ToLower(strings.TrimSpace(candidate.InsertText))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}

	sort.SliceStable(unique, func(i, j int) bool {
		left := unique[i]
		right := unique[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.InsertText < right.InsertText
	})

	if len(unique) > maxResults {
		return unique[:maxResults]
	}
	return unique
}

func (e *CompletionEngine) suggestAgentAndModelMentions(token string, req CompletionRequest) []CompletionSuggestion {
	query := strings.TrimSpace(token)
	hasAt := strings.HasPrefix(query, "@")
	query = strings.TrimPrefix(query, "@")
	query = strings.ToLower(strings.TrimSpace(query))

	out := make([]CompletionSuggestion, 0, len(req.AgentTargets)+len(req.ModelTargets))
	for _, agent := range req.AgentTargets {
		target := normalizeCompletionToken(agent)
		if target == "" {
			continue
		}
		candidate := "run-agent-" + target
		score := fuzzyScore(candidate, query)
		if score < 0 {
			continue
		}
		insert := "@" + candidate
		if hasAt {
			insert = "@" + candidate
		}
		out = append(out, CompletionSuggestion{
			Kind:       CompletionKindAgent,
			Label:      insert,
			InsertText: insert,
			Score:      score + 150,
		})
	}
	for _, model := range req.ModelTargets {
		target := normalizeCompletionToken(model)
		if target == "" {
			continue
		}
		candidate := "ask-" + target
		score := fuzzyScore(candidate, query)
		if score < 0 {
			continue
		}
		insert := "@" + candidate
		if hasAt {
			insert = "@" + candidate
		}
		out = append(out, CompletionSuggestion{
			Kind:       CompletionKindModel,
			Label:      insert,
			InsertText: insert,
			Score:      score + 140,
		})
	}
	return out
}

func (e *CompletionEngine) suggestSlashCommands(token string, req CompletionRequest) []CompletionSuggestion {
	if len(req.SlashCommands) == 0 {
		return nil
	}
	query := strings.TrimSpace(token)
	if !strings.HasPrefix(query, "/") {
		return nil
	}
	query = strings.ToLower(strings.TrimPrefix(query, "/"))
	out := make([]CompletionSuggestion, 0, len(req.SlashCommands))
	for _, cmd := range req.SlashCommands {
		name := normalizeCompletionToken(cmd)
		if name == "" {
			continue
		}
		score := fuzzyScore(name, query)
		if score < 0 {
			continue
		}
		insert := "/" + name
		out = append(out, CompletionSuggestion{
			Kind:       CompletionKindSlash,
			Label:      insert,
			InsertText: insert,
			Score:      score + 120,
		})
	}
	return out
}

func (e *CompletionEngine) suggestFiles(token string, req CompletionRequest) []CompletionSuggestion {
	query := strings.TrimSpace(token)
	hasAt := strings.HasPrefix(query, "@")
	pathQuery := strings.TrimPrefix(query, "@")
	if !(hasAt || strings.Contains(pathQuery, "/") || strings.HasPrefix(pathQuery, ".") || strings.HasPrefix(pathQuery, "~")) {
		return nil
	}

	workingDir := strings.TrimSpace(req.WorkingDir)
	if workingDir == "" {
		workingDir = "."
	}
	baseDir := workingDir
	prefix := ""
	fileNameQuery := pathQuery
	if strings.Contains(pathQuery, "/") {
		base := filepath.Dir(pathQuery)
		if strings.TrimSpace(base) != "" && base != "." {
			baseDir = filepath.Join(workingDir, base)
			prefix = filepath.ToSlash(base) + "/"
		}
		fileNameQuery = filepath.Base(pathQuery)
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}
	out := make([]CompletionSuggestion, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		candidate := prefix + name
		score := fuzzyScore(strings.ToLower(name), strings.ToLower(fileNameQuery))
		if score < 0 {
			continue
		}
		insert := candidate
		if hasAt {
			insert = "@" + candidate
		}
		out = append(out, CompletionSuggestion{
			Kind:       CompletionKindFile,
			Label:      insert,
			InsertText: insert,
			Score:      score + 100,
		})
	}
	return out
}

func (e *CompletionEngine) suggestUnixCommands(token string, req CompletionRequest) []CompletionSuggestion {
	query := strings.TrimSpace(token)
	if strings.HasPrefix(query, "/") || strings.HasPrefix(query, "@") {
		return nil
	}
	query = strings.TrimPrefix(query, "!")
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	commands := e.discoverPATHCommands(req.Env)
	if len(commands) == 0 {
		return nil
	}
	out := make([]CompletionSuggestion, 0, len(commands))
	for _, command := range commands {
		score := fuzzyScore(command, query)
		if score < 0 {
			continue
		}
		score += unixCommandPriorityBoost(command)
		out = append(out, CompletionSuggestion{
			Kind:       CompletionKindCommand,
			Label:      command,
			InsertText: command,
			Score:      score + 90,
		})
	}
	return out
}

func (e *CompletionEngine) discoverPATHCommands(env map[string]string) []string {
	if !isUnixLikePlatform(runtime.GOOS) {
		return nil
	}
	pathEnv := strings.TrimSpace(env["PATH"])
	if pathEnv == "" {
		pathEnv = strings.TrimSpace(os.Getenv("PATH"))
	}
	if pathEnv == "" {
		return nil
	}
	cacheKey := runtime.GOOS + "|" + pathEnv

	e.mu.Lock()
	if entry, ok := e.commandCache[cacheKey]; ok {
		if time.Since(entry.loadedAt) < e.cacheTTL {
			cached := append([]string{}, entry.commands...)
			e.mu.Unlock()
			return cached
		}
	}
	e.mu.Unlock()

	seen := map[string]struct{}{}
	commands := make([]string, 0, 256)
	dirs := filepath.SplitList(pathEnv)
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0o111 == 0 {
				continue
			}
			name := normalizeCompletionToken(entry.Name())
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			commands = append(commands, name)
		}
	}
	sort.Strings(commands)

	e.mu.Lock()
	e.commandCache[cacheKey] = pathCommandCacheEntry{
		commands: append([]string{}, commands...),
		loadedAt: time.Now(),
	}
	e.mu.Unlock()
	return commands
}

func unixCommandPriorityBoost(command string) int {
	switch command {
	case "git", "rg", "ls", "cat", "grep", "go", "node", "pnpm", "npm", "bash", "zsh", "sed", "awk", "make":
		return 80
	default:
		return 0
	}
}

func lastToken(input string) string {
	trimmed := strings.TrimRight(input, " \t\r\n")
	if trimmed == "" {
		return ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func normalizeCompletionToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func fuzzyScore(candidate string, query string) int {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	query = strings.ToLower(strings.TrimSpace(query))
	if candidate == "" {
		return -1
	}
	if query == "" {
		return 1
	}
	if strings.HasPrefix(candidate, query) {
		return 1000 - len(candidate)
	}
	if strings.Contains(candidate, query) {
		return 500 - len(candidate)
	}

	score := 0
	cIdx := 0
	for qIdx := 0; qIdx < len(query); qIdx++ {
		qChar := query[qIdx]
		found := false
		for cIdx < len(candidate) {
			if candidate[cIdx] == qChar {
				score += 10
				if cIdx > 0 && qIdx > 0 && candidate[cIdx-1] == query[qIdx-1] {
					score += 5
				}
				cIdx++
				found = true
				break
			}
			cIdx++
		}
		if !found {
			return -1
		}
	}
	return score - (len(candidate) - len(query))
}

func isUnixLikePlatform(goos string) bool {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "linux", "darwin":
		return true
	default:
		return false
	}
}
