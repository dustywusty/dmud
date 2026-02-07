package persistence

import (
	"context"
	"encoding/json"
	"sync"
)

type MemoryStore struct {
	mu      sync.RWMutex
	players map[string]*PlayerState
	world   *WorldState
	admins  map[string]bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		players: make(map[string]*PlayerState),
		admins:  make(map[string]bool),
	}
}

func (m *MemoryStore) LoadPlayer(_ context.Context, key string) (*PlayerState, error) {
	m.mu.RLock()
	state, ok := m.players[key]
	m.mu.RUnlock()
	if !ok || state == nil {
		return nil, nil
	}

	clone := &PlayerState{}
	if err := cloneFromState(state, clone); err != nil {
		return nil, err
	}
	return clone, nil
}

func (m *MemoryStore) SavePlayer(_ context.Context, key string, state *PlayerState) error {
	if state == nil {
		return nil
	}
	clone := &PlayerState{}
	if err := cloneFromState(state, clone); err != nil {
		return err
	}
	m.mu.Lock()
	m.players[key] = clone
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) DeletePlayer(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.players, key)
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) LoadWorld(_ context.Context) (*WorldState, error) {
	m.mu.RLock()
	state := m.world
	m.mu.RUnlock()
	if state == nil {
		return nil, nil
	}
	clone := &WorldState{}
	if err := cloneFromWorld(state, clone); err != nil {
		return nil, err
	}
	return clone, nil
}

func (m *MemoryStore) SaveWorld(_ context.Context, state *WorldState) error {
	if state == nil {
		return nil
	}
	clone := &WorldState{}
	if err := cloneFromWorld(state, clone); err != nil {
		return err
	}
	m.mu.Lock()
	m.world = clone
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) IsAdmin(_ context.Context, key string) (bool, error) {
	if key == "" {
		return false, nil
	}
	m.mu.RLock()
	isAdmin := m.admins[key]
	m.mu.RUnlock()
	return isAdmin, nil
}

func (m *MemoryStore) SetAdmin(_ context.Context, key string, isAdmin bool) error {
	if key == "" {
		return nil
	}
	m.mu.Lock()
	if isAdmin {
		m.admins[key] = true
	} else {
		delete(m.admins, key)
	}
	m.mu.Unlock()
	return nil
}

func cloneFromState(src *PlayerState, dest *PlayerState) error {
	payload, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dest)
}

func cloneFromWorld(src *WorldState, dest *WorldState) error {
	payload, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dest)
}
