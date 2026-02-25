package tui

import (
	"fmt"
	"io"
	"strings"

	"goyais/services/hub/internal/agentcore/protocol"
)

type EventRenderer struct {
	stdout io.Writer
	stderr io.Writer
}

func NewEventRenderer(stdout io.Writer, stderr io.Writer) EventRenderer {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return EventRenderer{
		stdout: stdout,
		stderr: stderr,
	}
}

func (r EventRenderer) Render(event protocol.RunEvent) error {
	switch event.Type {
	case protocol.RunEventTypeRunOutputDelta:
		if text := payloadText(event.Payload, "delta", "output", "content"); text != "" {
			_, err := io.WriteString(r.stdout, text)
			return err
		}
	case protocol.RunEventTypeRunApprovalNeeded:
		_, err := io.WriteString(r.stdout, "[approval required]\n")
		return err
	case protocol.RunEventTypeRunFailed:
		message := payloadText(event.Payload, "message", "error")
		if message == "" {
			message = "run failed"
		}
		_, err := fmt.Fprintf(r.stderr, "error: %s\n", message)
		return err
	case protocol.RunEventTypeRunCancelled:
		_, err := io.WriteString(r.stderr, "run cancelled\n")
		return err
	case protocol.RunEventTypeRunCompleted:
		_, err := io.WriteString(r.stdout, "\n")
		return err
	}
	return nil
}

func payloadText(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, exists := payload[key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			return text
		}
	}
	return ""
}
