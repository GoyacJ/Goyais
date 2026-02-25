package config

import (
	"errors"
	"fmt"
	"strings"
)

type SessionMode string

const (
	SessionModeAgent SessionMode = "agent"
	SessionModePlan  SessionMode = "plan"
)

type ResolvedConfig struct {
	GlobalPath   string
	ProjectPath  string
	SessionMode  SessionMode
	DefaultModel string
	Env          map[string]string
}

func (c ResolvedConfig) Validate() error {
	mode := SessionMode(strings.TrimSpace(string(c.SessionMode)))
	if mode == "" {
		return errors.New("session_mode is required")
	}
	if mode != SessionModeAgent && mode != SessionModePlan {
		return fmt.Errorf("session_mode %q is not supported", mode)
	}
	if strings.TrimSpace(c.DefaultModel) == "" {
		return errors.New("default_model is required")
	}
	return nil
}

type Provider interface {
	Load(globalPath string, projectPath string, env map[string]string) (ResolvedConfig, error)
}

type StaticProvider struct {
	Config ResolvedConfig
	Err    error
}

func (p StaticProvider) Load(globalPath string, projectPath string, env map[string]string) (ResolvedConfig, error) {
	if p.Err != nil {
		return ResolvedConfig{}, p.Err
	}

	out := p.Config
	out.GlobalPath = strings.TrimSpace(globalPath)
	out.ProjectPath = strings.TrimSpace(projectPath)
	if len(env) > 0 {
		out.Env = cloneStringMap(env)
	} else {
		out.Env = cloneStringMap(out.Env)
	}

	if err := out.Validate(); err != nil {
		return ResolvedConfig{}, err
	}
	return out, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
