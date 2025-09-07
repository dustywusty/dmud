package components

import "sync"

type Inventory struct {
	sync.RWMutex
	Items []string
}
