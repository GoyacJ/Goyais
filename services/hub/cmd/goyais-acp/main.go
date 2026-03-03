package main

import (
	"context"
	"fmt"
	"os"

	acpadapter "goyais/services/hub/internal/agent/adapters/acp"
	runtimebridge "goyais/services/hub/internal/agent/adapters/runtimebridge"
	"goyais/services/hub/internal/agent/runtime/loop"
)

func main() {
	engine := loop.NewEngine(nil)
	eventSink := runtimebridge.CLIProjector{
		Projector: runtimebridge.NewProjector(runtimebridge.ProjectorOptions{
			Store: runtimebridge.NewMemoryEventStore(),
		}),
	}
	peer := acpadapter.NewPeer()
	_ = acpadapter.NewServer(peer, acpadapter.ServerOptions{
		Bridge: acpadapter.NewBridgeWithOptions(engine, nil, acpadapter.BridgeOptions{
			Projector: eventSink,
		}),
	})

	transport := acpadapter.NewStdioTransport(peer, acpadapter.StdioTransportOptions{
		Input: os.Stdin,
		WriteLine: func(line string) error {
			_, err := fmt.Fprintln(os.Stdout, line)
			return err
		},
	})

	if err := transport.Start(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "goyais-acp transport error: %v\n", err)
		os.Exit(1)
	}
}
