package persistence

import "context"

type NoopStore struct{}

func (n *NoopStore) LoadPlayer(_ context.Context, _ string) (*PlayerState, error) {
	return nil, nil
}

func (n *NoopStore) SavePlayer(_ context.Context, _ string, _ *PlayerState) error {
	return nil
}

func (n *NoopStore) DeletePlayer(_ context.Context, _ string) error {
	return nil
}

func (n *NoopStore) LoadWorld(_ context.Context) (*WorldState, error) {
	return nil, nil
}

func (n *NoopStore) SaveWorld(_ context.Context, _ *WorldState) error {
	return nil
}

func (n *NoopStore) IsAdmin(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (n *NoopStore) SetAdmin(_ context.Context, _ string, _ bool) error {
	return nil
}
