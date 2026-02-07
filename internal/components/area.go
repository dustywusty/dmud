package components

import (
	"dmud/internal/util"
	"sync"

	"github.com/rs/zerolog/log"
)

type Exit struct {
	Direction string
	AreaID    string
	Area      *Area
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
}

func (a *Area) AddPlayer(p *Player) {
	log.Info().Msgf("Player added to area: %s", p.Name)

	a.Broadcast(util.TagMessage("STATUS", p.Name+" enters"))

	a.PlayersMutex.Lock()
	a.Players = append(a.Players, p)
	a.PlayersMutex.Unlock()
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
		npc.RLock()
		sameArea := npc.Area == a
		npc.RUnlock()
		return sameArea
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
