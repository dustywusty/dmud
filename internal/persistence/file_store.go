package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type FileStore struct {
	mu  sync.RWMutex
	dir string
}

func NewFileStore(dir string) (*FileStore, error) {
	if dir == "" {
		dir = "data"
	}
	playersDir := filepath.Join(dir, "players")
	if err := os.MkdirAll(playersDir, 0755); err != nil {
		return nil, err
	}
	return &FileStore{dir: dir}, nil
}

func (f *FileStore) LoadPlayer(_ context.Context, key string) (*PlayerState, error) {
	if key == "" {
		return nil, nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := os.ReadFile(f.playerPath(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state PlayerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (f *FileStore) SavePlayer(_ context.Context, key string, state *PlayerState) error {
	if key == "" || state == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.writeJSON(f.playerPath(key), state)
}

func (f *FileStore) DeletePlayer(_ context.Context, key string) error {
	if key == "" {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	err := os.Remove(f.playerPath(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (f *FileStore) LoadWorld(_ context.Context) (*WorldState, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := os.ReadFile(f.worldPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state WorldState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (f *FileStore) SaveWorld(_ context.Context, state *WorldState) error {
	if state == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.writeJSON(f.worldPath(), state)
}

func (f *FileStore) IsAdmin(_ context.Context, key string) (bool, error) {
	if key == "" {
		return false, nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()

	admins, _ := f.loadAdmins()
	return admins[key], nil
}

func (f *FileStore) SetAdmin(_ context.Context, key string, isAdmin bool) error {
	if key == "" {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	admins, _ := f.loadAdmins()
	if admins == nil {
		admins = make(map[string]bool)
	}
	if isAdmin {
		admins[key] = true
	} else {
		delete(admins, key)
	}
	return f.writeJSON(f.adminsPath(), admins)
}

// writeJSON atomically writes JSON data to a file using a temp file + rename.
func (f *FileStore) writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f *FileStore) loadAdmins() (map[string]bool, error) {
	data, err := os.ReadFile(f.adminsPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var admins map[string]bool
	err = json.Unmarshal(data, &admins)
	return admins, err
}

func (f *FileStore) playerPath(key string) string {
	return filepath.Join(f.dir, "players", key+".json")
}

func (f *FileStore) worldPath() string {
	return filepath.Join(f.dir, "world.json")
}

func (f *FileStore) adminsPath() string {
	return filepath.Join(f.dir, "admins.json")
}
