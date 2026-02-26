package stdio

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

type UserMessage struct {
	UUID    string
	Message map[string]any
	Raw     map[string]any
}

type ControlRequest struct {
	RequestID string
	Request   map[string]any
	Raw       map[string]any
}

type HandlerOptions struct {
	OnInterrupt      func()
	OnControlRequest func(req ControlRequest) (any, error)
}

type StructuredStdio struct {
	input  io.Reader
	output io.Writer
	opts   HandlerOptions

	mu      sync.Mutex
	started bool
	closed  bool

	userQueue chan UserMessage
	errQueue  chan error
}

func NewStructuredStdio(input io.Reader, output io.Writer, opts HandlerOptions) *StructuredStdio {
	return &StructuredStdio{
		input:     input,
		output:    output,
		opts:      opts,
		userQueue: make(chan UserMessage, 32),
		errQueue:  make(chan error, 1),
	}
}

func (s *StructuredStdio) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()

	go s.readLoop()
}

func (s *StructuredStdio) NextUserMessage(ctx context.Context) (UserMessage, error) {
	if s == nil {
		return UserMessage{}, errors.New("structured stdio is nil")
	}
	select {
	case <-ctx.Done():
		return UserMessage{}, ctx.Err()
	case msg, ok := <-s.userQueue:
		if !ok {
			return UserMessage{}, io.EOF
		}
		return msg, nil
	case err, ok := <-s.errQueue:
		if !ok || err == nil {
			return UserMessage{}, io.EOF
		}
		return UserMessage{}, err
	}
}

func (s *StructuredStdio) readLoop() {
	defer func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.userQueue)
	}()

	scanner := bufio.NewScanner(s.input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		raw := map[string]any{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		typeName, _ := raw["type"].(string)
		typeName = strings.TrimSpace(typeName)
		switch typeName {
		case "keep_alive":
			continue
		case "user":
			message, _ := raw["message"].(map[string]any)
			uuid, _ := raw["uuid"].(string)
			s.userQueue <- UserMessage{
				UUID:    strings.TrimSpace(uuid),
				Message: message,
				Raw:     raw,
			}
		case "control_request":
			s.handleControlRequest(raw)
		case "control_cancel_request":
			continue
		default:
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		select {
		case s.errQueue <- err:
		default:
		}
	}
}

func (s *StructuredStdio) handleControlRequest(raw map[string]any) {
	requestID, _ := raw["request_id"].(string)
	requestID = strings.TrimSpace(requestID)
	request, _ := raw["request"].(map[string]any)
	subtype, _ := request["subtype"].(string)
	subtype = strings.TrimSpace(subtype)

	if requestID == "" {
		return
	}
	if subtype == "" {
		_ = s.WriteControlResponseError(requestID, "Invalid control request (missing subtype)")
		return
	}

	if subtype == "interrupt" {
		if s.opts.OnInterrupt != nil {
			s.opts.OnInterrupt()
		}
		_ = s.WriteControlResponseSuccess(requestID, nil)
		return
	}

	if s.opts.OnControlRequest == nil {
		_ = s.WriteControlResponseError(requestID, fmt.Sprintf("Unsupported control request subtype: %s", subtype))
		return
	}

	response, err := s.opts.OnControlRequest(ControlRequest{
		RequestID: requestID,
		Request:   request,
		Raw:       raw,
	})
	if err != nil {
		_ = s.WriteControlResponseError(requestID, err.Error())
		return
	}
	_ = s.WriteControlResponseSuccess(requestID, response)
}

func (s *StructuredStdio) WriteLine(payload any) error {
	if s == nil || s.output == nil {
		return nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = io.WriteString(s.output, string(encoded)+"\n")
	return err
}

func (s *StructuredStdio) WriteControlResponseSuccess(requestID string, response any) error {
	payload := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": requestID,
		},
	}
	if response != nil {
		payload["response"].(map[string]any)["response"] = response
	}
	return s.WriteLine(payload)
}

func (s *StructuredStdio) WriteControlResponseError(requestID string, errText string) error {
	payload := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "error",
			"request_id": requestID,
			"error":      errText,
		},
	}
	return s.WriteLine(payload)
}
