package httpapi

import (
	"os/exec"
	"strings"
)

func isGitRepositoryPath(projectPath string) bool {
	normalizedPath := strings.TrimSpace(projectPath)
	if normalizedPath == "" {
		return false
	}
	output, err := exec.Command("git", "-C", normalizedPath, "rev-parse", "--is-inside-work-tree").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(output)), "true")
}
