package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type StdioTransport struct {
	peer      *Peer
	input     io.Reader
	writeLine func(line string) error
}

type StdioTransportOptions struct {
	Input     io.Reader
	WriteLine func(line string) error
}

func NewStdioTransport(peer *Peer, opts StdioTransportOptions) *StdioTransport {
	return &StdioTransport{
		peer:      peer,
		input:     opts.Input,
		writeLine: opts.WriteLine,
	}
}

func (t *StdioTransport) Start(ctx context.Context) error {
	if t == nil || t.peer == nil {
		return errors.New("transport peer is required")
	}
	if t.input == nil {
		return errors.New("transport input is required")
	}
	if t.writeLine == nil {
		return errors.New("transport writer is required")
	}

	t.peer.SetSend(t.writeLine)
	scanner := bufio.NewScanner(t.input)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var payload any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			_ = t.writeLine(string(mustMarshalJSON(buildErrorResponse(nil, -32700, "Parse error", nil))))
			continue
		}
		if err := t.peer.HandleIncoming(payload); err != nil {
			_ = t.writeLine(string(mustMarshalJSON(buildErrorResponse(nil, -32603, err.Error(), nil))))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func mustMarshalJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"Internal error"}}`)
	}
	return encoded
}
