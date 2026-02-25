package cli

import (
	"errors"
	"fmt"
	"strings"
)

type Options struct {
	HelpLite bool
	Help     bool
	Version  bool
	Print    bool
	CWD      string
	Prompt   string
}

func ParseOptions(args []string) (Options, error) {
	var opts Options

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]

		switch {
		case arg == "--help-lite":
			opts.HelpLite = true
		case arg == "--help" || arg == "-h":
			opts.Help = true
		case arg == "--version" || arg == "-v":
			opts.Version = true
		case arg == "--print" || arg == "-p":
			opts.Print = true
		case arg == "--cwd" || arg == "-c":
			if idx+1 >= len(args) {
				return Options{}, errors.New("--cwd requires a path argument")
			}
			idx++
			opts.CWD = strings.TrimSpace(args[idx])
		case strings.HasPrefix(arg, "--cwd="):
			opts.CWD = strings.TrimSpace(strings.TrimPrefix(arg, "--cwd="))
		case strings.HasPrefix(arg, "-c="):
			opts.CWD = strings.TrimSpace(strings.TrimPrefix(arg, "-c="))
		case arg == "--":
			opts.Prompt = strings.TrimSpace(strings.Join(args[idx+1:], " "))
			return opts, nil
		case strings.HasPrefix(arg, "-"):
			return Options{}, fmt.Errorf("unknown option %q", arg)
		default:
			opts.Prompt = strings.TrimSpace(strings.Join(args[idx:], " "))
			return opts, nil
		}
	}

	return opts, nil
}
