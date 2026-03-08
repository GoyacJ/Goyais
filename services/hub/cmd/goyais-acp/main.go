package main

import (
	"context"
	"fmt"
	"os"

	acpadapter "goyais/services/hub/internal/agent/adapters/acp"
	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
	"goyais/services/hub/internal/agent/runtime/loop"
	runtimesession "goyais/services/hub/internal/agent/runtime/session"
	slashruntime "goyais/services/hub/internal/agent/runtime/slash"
	checkpointtool "goyais/services/hub/internal/agent/tools/checkpoint"
)

func main() {
	loopEngine := loop.NewEngine(nil)
	lifecycle := runtimesession.NewManager(runtimesession.Dependencies{
		CheckpointStore: checkpointtool.NewStore(""),
	})
	engine := runtimesession.NewTrackingEngine(loopEngine, lifecycle, runtimesession.TrackingEngineOptions{})
	lifecycle.SetStarter(engine)
	checkpoints := agenthttpapi.NewLifecycleCheckpointBridge(lifecycle)
	commandBus := slashruntime.NewBus(slashruntime.NewLoopContextResolver(loopEngine))
	peer := acpadapter.NewPeer()
	_ = acpadapter.NewServer(peer, acpadapter.ServerOptions{
		Bridge:            acpadapter.NewBridge(engine, commandBus),
		Lifecycle:         lifecycle,
		CheckpointService: checkpoints,
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
