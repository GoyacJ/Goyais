package acp

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

const jsonRPCVersion = "2.0"

type JsonRPCID = any

type JsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e JsonRPCError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("json-rpc error %d", e.Code)
	}
	return e.Message
}

type methodHandler func(params any) (any, error)

type pendingRequest struct {
	resolve   func(any)
	reject    func(error)
	timeoutID *time.Timer
}

type Peer struct {
	mu       sync.Mutex
	handlers map[string]methodHandler
	pending  map[string]pendingRequest
	nextID   int64
	sendLine func(line string) error
}

func NewPeer() *Peer {
	return &Peer{
		handlers: map[string]methodHandler{},
		pending:  map[string]pendingRequest{},
		nextID:   1,
	}
}

func (p *Peer) SetSend(send func(line string) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sendLine = send
}

func (p *Peer) RegisterMethod(method string, handler func(params any) (any, error)) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[method] = handler
}

func (p *Peer) SendNotification(method string, params any) error {
	payload := map[string]any{
		"jsonrpc": jsonRPCVersion,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	return p.sendRaw(payload)
}

func (p *Peer) SendRequest(method string, params any, timeout time.Duration) (any, error) {
	if p == nil {
		return nil, errors.New("peer is nil")
	}

	p.mu.Lock()
	id := p.nextID
	p.nextID++

	resultCh := make(chan any, 1)
	errCh := make(chan error, 1)
	entry := pendingRequest{
		resolve: func(value any) {
			resultCh <- value
		},
		reject: func(err error) {
			errCh <- err
		},
	}
	if timeout > 0 {
		entry.timeoutID = time.AfterFunc(timeout, func() {
			p.mu.Lock()
			delete(p.pending, stringifyID(id))
			p.mu.Unlock()
			errCh <- JsonRPCError{Code: -32000, Message: fmt.Sprintf("Request timed out: %s", method)}
		})
	}
	p.pending[stringifyID(id)] = entry
	p.mu.Unlock()

	payload := map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	if err := p.sendRaw(payload); err != nil {
		p.mu.Lock()
		delete(p.pending, stringifyID(id))
		p.mu.Unlock()
		return nil, err
	}

	select {
	case value := <-resultCh:
		return value, nil
	case err := <-errCh:
		return nil, err
	}
}

func (p *Peer) HandleIncoming(payload any) error {
	if payload == nil {
		return p.sendRaw(buildErrorResponse(nil, -32600, "Invalid Request", nil))
	}

	if batch, ok := payload.([]any); ok {
		responses := make([]any, 0, len(batch))
		for _, item := range batch {
			response, handled := p.handleIncomingOne(item)
			if handled && response != nil {
				responses = append(responses, response)
			}
		}
		if len(responses) == 0 {
			return nil
		}
		return p.sendRaw(responses)
	}

	response, handled := p.handleIncomingOne(payload)
	if !handled || response == nil {
		return nil
	}
	return p.sendRaw(response)
}

func (p *Peer) handleIncomingOne(payload any) (any, bool) {
	obj, ok := payload.(map[string]any)
	if !ok {
		return buildErrorResponse(nil, -32600, "Invalid Request", nil), true
	}

	if version, _ := obj["jsonrpc"].(string); version != jsonRPCVersion {
		return buildErrorResponse(normalizeID(obj["id"]), -32600, "Invalid Request", nil), true
	}

	method, hasMethod := obj["method"].(string)
	id, hasID := normalizeResponseID(obj["id"])

	if !hasMethod && hasID && (obj["result"] != nil || obj["error"] != nil) {
		p.handleResponse(obj)
		return nil, false
	}
	if !hasMethod || method == "" {
		return buildErrorResponse(normalizeID(obj["id"]), -32600, "Invalid Request", nil), true
	}

	params := obj["params"]
	if !hasID {
		handler, ok := p.getHandler(method)
		if !ok {
			return nil, false
		}
		_, _ = handler(params)
		return nil, false
	}

	handler, ok := p.getHandler(method)
	if !ok {
		return buildErrorResponse(id, -32601, fmt.Sprintf("Method not found: %s", method), nil), true
	}

	result, err := handler(params)
	if err != nil {
		var rpcErr JsonRPCError
		if errors.As(err, &rpcErr) {
			return buildErrorResponse(id, rpcErr.Code, rpcErr.Message, rpcErr.Data), true
		}
		return buildErrorResponse(id, -32603, err.Error(), nil), true
	}
	if result == nil {
		result = map[string]any{}
	}
	return map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"result":  result,
	}, true
}

func (p *Peer) handleResponse(msg map[string]any) {
	id := normalizeID(msg["id"])
	if id == nil {
		return
	}

	p.mu.Lock()
	entry, ok := p.pending[stringifyID(id)]
	if ok {
		delete(p.pending, stringifyID(id))
	}
	p.mu.Unlock()
	if !ok {
		return
	}
	if entry.timeoutID != nil {
		entry.timeoutID.Stop()
	}

	if errorValue, hasError := msg["error"]; hasError && errorValue != nil {
		errObj, _ := errorValue.(map[string]any)
		code := asInt(errObj["code"], -32603)
		message, _ := errObj["message"].(string)
		if message == "" {
			message = "Unknown error"
		}
		entry.reject(JsonRPCError{
			Code:    code,
			Message: message,
			Data:    errObj["data"],
		})
		return
	}

	entry.resolve(msg["result"])
}

func (p *Peer) getHandler(method string) (methodHandler, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	handler, ok := p.handlers[method]
	return handler, ok
}

func (p *Peer) sendRaw(payload any) error {
	p.mu.Lock()
	send := p.sendLine
	p.mu.Unlock()

	if send == nil {
		return errors.New("json-rpc send is not configured")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return send(string(encoded))
}

func buildErrorResponse(id any, code int, message string, data any) map[string]any {
	errObj := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		errObj["data"] = data
	}
	return map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"error":   errObj,
	}
}

func normalizeResponseID(value any) (any, bool) {
	switch value.(type) {
	case string, float64, float32, int, int64, int32, uint, uint64, uint32:
		return normalizeID(value), true
	default:
		return nil, false
	}
}

func normalizeID(value any) any {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	case int32:
		return int64(typed)
	case uint:
		return int64(typed)
	case uint64:
		return int64(typed)
	case uint32:
		return int64(typed)
	default:
		return nil
	}
}

func stringifyID(id any) string {
	switch typed := id.(type) {
	case string:
		return typed
	case int64:
		return fmt.Sprintf("%d", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func asInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}
