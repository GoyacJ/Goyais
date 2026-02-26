package main

import (
	"context"
	"fmt"
	"os"

	"goyais/services/hub/internal/agentcore/acp"
	"goyais/services/hub/internal/agentcore/config"
	"goyais/services/hub/internal/agentcore/runtime"
)

func main() {
	guard := acp.InstallStdoutGuard()
	defer guard.Restore()

	peer := acp.NewPeer()
	_ = acp.NewAgent(peer, acp.AgentOptions{
		ConfigProvider: config.StaticProvider{
			Config: config.ResolvedConfig{
				SessionMode:  config.SessionModeAgent,
				DefaultModel: "gpt-5",
			},
		},
		Engine: runtime.NewLocalEngine(),
	})

	transport := acp.NewStdioTransport(peer, acp.StdioTransportOptions{
		Input: os.Stdin,
		WriteLine: func(line string) error {
			return guard.WriteLine(line)
		},
	})

	if err := transport.Start(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "goyais-acp transport error: %v\n", err)
		os.Exit(1)
	}
}
