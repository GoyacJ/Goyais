package input

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrImagePasteUnsupportedPlatform = errors.New("image paste is not supported on this platform")
	ErrImagePasteUnavailable         = errors.New("clipboard does not contain an image")
)

type ClipboardImageStore struct {
	nextID int
	items  map[string]string
}

func NewClipboardImageStore() *ClipboardImageStore {
	return &ClipboardImageStore{
		nextID: 1,
		items:  map[string]string{},
	}
}

func (s *ClipboardImageStore) PasteFromClipboard(workingDir string, env map[string]string) (string, error) {
	path, err := SaveClipboardImage(workingDir, env)
	if err != nil {
		return "", err
	}
	placeholder := fmt.Sprintf("[Image #%d]", s.nextID)
	s.nextID++
	s.items[placeholder] = path
	return placeholder, nil
}

func (s *ClipboardImageStore) Lookup(placeholder string) (string, bool) {
	path, ok := s.items[placeholder]
	return path, ok
}

func SaveClipboardImage(workingDir string, env map[string]string) (string, error) {
	platform := resolveImagePastePlatform(env)
	return saveClipboardImage(platform, workingDir, env, exec.LookPath, func(cmd *exec.Cmd) error {
		return cmd.Run()
	})
}

func saveClipboardImage(
	platform string,
	workingDir string,
	env map[string]string,
	lookPathFn func(string) (string, error),
	runFn func(*exec.Cmd) error,
) (string, error) {
	if strings.TrimSpace(platform) != "darwin" {
		return "", ErrImagePasteUnsupportedPlatform
	}

	bin := firstNonEmptyString(env["GOYAIS_PNGPASTE_BIN"], "pngpaste")
	resolvedBin := strings.TrimSpace(bin)
	if !filepath.IsAbs(resolvedBin) {
		path, err := lookPathFn(resolvedBin)
		if err != nil {
			return "", ErrImagePasteUnavailable
		}
		resolvedBin = path
	}

	tempRoot := strings.TrimSpace(workingDir)
	if tempRoot == "" {
		tempRoot = os.TempDir()
	}
	if _, err := os.Stat(tempRoot); err != nil {
		tempRoot = os.TempDir()
	}

	file, err := os.CreateTemp(tempRoot, "goyais-pasted-image-*.png")
	if err != nil {
		return "", err
	}
	tempPath := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}

	cmd := exec.Command(resolvedBin, tempPath)
	cmd.Env = mergeImagePasteEnv(os.Environ(), env)
	if err := runFn(cmd); err != nil {
		_ = os.Remove(tempPath)
		return "", ErrImagePasteUnavailable
	}

	stat, err := os.Stat(tempPath)
	if err != nil || stat.Size() == 0 {
		_ = os.Remove(tempPath)
		return "", ErrImagePasteUnavailable
	}

	return tempPath, nil
}

func resolveImagePastePlatform(env map[string]string) string {
	return strings.ToLower(strings.TrimSpace(firstNonEmptyString(
		env["GOYAIS_IMAGE_PASTE_PLATFORM"],
		runtime.GOOS,
	)))
}

func mergeImagePasteEnv(base []string, override map[string]string) []string {
	if len(override) == 0 {
		return base
	}
	merged := append([]string{}, base...)
	indexByKey := map[string]int{}
	for idx, kv := range merged {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		indexByKey[kv[:eq]] = idx
	}
	for key, value := range override {
		if idx, ok := indexByKey[key]; ok {
			merged[idx] = key + "=" + value
		} else {
			merged = append(merged, key+"="+value)
		}
	}
	return merged
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
