package eventbus

import "context"

type MemoryProvider struct{}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{}
}

func (p *MemoryProvider) Publish(context.Context, string, Message) error {
	return nil
}

func (p *MemoryProvider) Ping(context.Context) error {
	return nil
}

func (p *MemoryProvider) Close() error {
	return nil
}

func (p *MemoryProvider) Name() string {
	return "memory"
}
