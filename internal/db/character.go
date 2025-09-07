package db

import (
	"context"
	"errors"
	"sync"
)

type Character struct {
	ID        string
	Name      string
	RoomID    string
	Health    int
	MaxHealth int
	Inventory []string
}

var (
	characters = make(map[string]*Character)
	mu         sync.RWMutex
)

func LoadCharacter(ctx context.Context, id string) (*Character, error) {
	mu.RLock()
	defer mu.RUnlock()

	if ch, ok := characters[id]; ok {
		copy := *ch
		return &copy, nil
	}
	return nil, errors.New("character not found")
}

func SaveCharacter(ctx context.Context, ch *Character) error {
	mu.Lock()
	defer mu.Unlock()

	copy := *ch
	characters[ch.ID] = &copy
	return nil
}
