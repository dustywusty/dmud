package components

import (
	"dmud/internal/util"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
)

type Exit struct {
	Direction string
	AreaID    string
	Area      *Area
	MaxSize   Size // largest creature that fits; 0 = no limit
}

type Area struct {
	X int
	Y int
	Z int

	Region      string
	Description string
	Exits       []Exit
	Players     []*Player
	Items       []*Item

	PlayersMutex sync.RWMutex
	ItemsMutex   sync.RWMutex

	// dirty is set whenever the room's contents change; the game loop flushes
	// dirty areas as room.contents events a few times a second.
	dirty atomic.Bool
}

// MarkDirty flags the room's contents as changed.
func (a *Area) MarkDirty() { a.dirty.Store(true) }

// TakeDirty atomically reads and clears the dirty flag.
func (a *Area) TakeDirty() bool { return a.dirty.Swap(false) }

// roomContentsEvent is the structured "what's in this room" push.
type roomContentsEvent struct {
	Type    string   `json:"type"`
	Players []string `json:"players"`
	NPCs    []string `json:"npcs"`
	Items   []string `json:"items"`
	Corpses []string `json:"corpses"`
}

// BroadcastContents pushes a room.contents event to each tag-capable player in
// the area (each sees the others, not themselves), so clients can keep a live
// "who/what is here" list without anyone having to look.
func (a *Area) BroadcastContents(w WorldLike) {
	a.PlayersMutex.RLock()
	players := make([]*Player, len(a.Players))
	copy(players, a.Players)
	a.PlayersMutex.RUnlock()

	var npcs []string
	for _, npc := range a.GetNPCs(w) {
		npcs = append(npcs, npc.Name)
	}
	var corpses []string
	for _, c := range a.GetCorpses(w) {
		corpses = append(corpses, c.GetDescription())
	}
	var items []string
	for _, it := range a.GetItems() {
		if it.Stackable && it.Quantity > 1 {
			items = append(items, fmt.Sprintf("%s x%d", it.Name, it.Quantity))
		} else {
			items = append(items, it.Name)
		}
	}

	for _, p := range players {
		if p.Client == nil || !p.Client.SupportsTags() {
			continue
		}
		others := make([]string, 0, len(players))
		for _, q := range players {
			if q != p {
				others = append(others, q.Name)
			}
		}
		ev := roomContentsEvent{Type: "room.contents", Players: others, NPCs: npcs, Items: items, Corpses: corpses}
		if data, err := json.Marshal(ev); err == nil {
			p.Broadcast(util.TagMessage("EVENT", string(data)))
		}
	}
}

func (a *Area) AddPlayer(p *Player) {
	log.Info().Msgf("Player added to area: %s", p.Name)

	a.Broadcast(util.TagMessage("STATUS", p.Name+" enters"))

	a.PlayersMutex.Lock()
	a.Players = append(a.Players, p)
	a.PlayersMutex.Unlock()

	a.MarkDirty()
}

func (a *Area) GetExit(direction string) *Exit {
	for i := range a.Exits {
		exit := &a.Exits[i]
		if exit.Direction == direction {
			return exit
		}
	}
	return nil
}

func (a *Area) GetNPCs(w WorldLike) []*NPC {
	var npcs []*NPC

	entities, err := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		npc, ok := i.(*NPC)
		if !ok {
			return false
		}
		return npc.Area == a
	})

	if err != nil {
		return npcs
	}

	for _, entity := range entities {
		npcComponent, err := w.GetComponent(entity.GetID(), "NPC")
		if err == nil {
			if npc, ok := npcComponent.(*NPC); ok {
				npcs = append(npcs, npc)
			}
		}
	}

	return npcs
}

func (a *Area) GetCorpses(w WorldLike) []*Corpse {
	var corpses []*Corpse

	entities, err := w.FindEntitiesByComponentPredicate("Corpse", func(i interface{}) bool {
		corpse, ok := i.(*Corpse)
		return ok && corpse.Area == a
	})

	if err != nil {
		return corpses
	}

	for _, entity := range entities {
		corpseComponent, err := w.GetComponent(entity.GetID(), "Corpse")
		if err == nil {
			if corpse, ok := corpseComponent.(*Corpse); ok {
				corpses = append(corpses, corpse)
			}
		}
	}

	return corpses
}

func (a *Area) AddItem(item *Item) {
	if item == nil {
		return
	}
	defer a.MarkDirty()
	a.ItemsMutex.Lock()
	defer a.ItemsMutex.Unlock()

	if item.Stackable {
		for _, existing := range a.Items {
			existing.Lock()
			if existing.ID == item.ID {
				existing.Quantity += item.Quantity
				existing.Unlock()
				return
			}
			existing.Unlock()
		}
	}

	a.Items = append(a.Items, item)
}

func (a *Area) RemoveItem(itemID string, quantity int) *Item {
	a.ItemsMutex.Lock()
	defer a.ItemsMutex.Unlock()
	defer a.MarkDirty()

	for i, item := range a.Items {
		item.Lock()
		if item.ID == itemID {
			if item.Stackable && item.Quantity > quantity {
				item.Quantity -= quantity
				removed := &Item{
					ID:          item.ID,
					Name:        item.Name,
					Description: item.Description,
					Type:        item.Type,
					Value:       item.Value,
					Stackable:   item.Stackable,
					Quantity:    quantity,
				}
				item.Unlock()
				return removed
			}
			removed := &Item{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				Type:        item.Type,
				Value:       item.Value,
				Stackable:   item.Stackable,
				Quantity:    item.Quantity,
			}
			item.Unlock()
			a.Items = append(a.Items[:i], a.Items[i+1:]...)
			return removed
		}
		item.Unlock()
	}

	return nil
}

func (a *Area) GetItems() []*Item {
	a.ItemsMutex.RLock()
	defer a.ItemsMutex.RUnlock()

	items := make([]*Item, len(a.Items))
	for i, item := range a.Items {
		items[i] = item.Clone()
	}
	return items
}

func (a *Area) GetPlayer(name string) *Player {
	a.PlayersMutex.RLock()
	defer a.PlayersMutex.RUnlock()

	for _, player := range a.Players {
		if player.Name == name {
			return player
		}
	}
	return nil
}

func (a *Area) Broadcast(msg string, exclude ...*Player) {
	a.PlayersMutex.Lock()
	defer a.PlayersMutex.Unlock()

	if len(a.Players) == 0 {
		return
	}

	for _, player := range a.Players {
		if !contains(exclude, player) {
			player.Broadcast(msg)
		}
	}
}

// BroadcastChat delivers a chat line to everyone in the area (minus excluded
// players): the structured event frame to tag-capable clients, the plain text to
// the rest. Snapshots under a read lock, then sends outside it.
func (a *Area) BroadcastChat(event, plain string, exclude ...*Player) {
	a.PlayersMutex.RLock()
	players := make([]*Player, 0, len(a.Players))
	for _, p := range a.Players {
		if !contains(exclude, p) {
			players = append(players, p)
		}
	}
	a.PlayersMutex.RUnlock()

	for _, p := range players {
		if p.Client != nil && p.Client.SupportsTags() {
			p.Broadcast(event)
		} else {
			p.Broadcast(plain)
		}
	}
}

func (a *Area) RemovePlayer(p *Player) {
	a.PlayersMutex.Lock()
	removed := false
	for i, player := range a.Players {
		if player == p {
			a.Players = append(a.Players[:i], a.Players[i+1:]...)
			removed = true
			break
		}
	}
	a.PlayersMutex.Unlock()

	if removed {
		a.Broadcast(util.TagMessage("STATUS", p.Name+" leaves"))
		a.MarkDirty()
	}
}

func contains(players []*Player, player *Player) bool {
	for _, p := range players {
		if p == player {
			return true
		}
	}
	return false
}
