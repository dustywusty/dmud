package components

import (
	"dmud/internal/common"
	"dmud/internal/util"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// charVitals is the structured push that mirrors the STATE| frame.
type charVitals struct {
	Type    string         `json:"type"`
	HP      int            `json:"hp"`
	MaxHP   int            `json:"max_hp"`
	EP      int            `json:"ep"`     // endurance (current)
	MaxEP   int            `json:"max_ep"` // endurance (max)
	Level   int            `json:"level"`
	XP      int            `json:"xp"`
	ReqXP   int            `json:"req_xp"`
	Area    string         `json:"area"`
	Gold    int            `json:"gold"`
	Effects []string       `json:"effects,omitempty"` // names only (legacy)
	Fx      []EffectView   `json:"fx,omitempty"`      // structured: kind + remaining + magnitude
	Mount   string         `json:"mount,omitempty"`
	Stats   map[string]int `json:"stats,omitempty"`
	Race    string         `json:"race,omitempty"`
}

type Player struct {
	sync.RWMutex

	Area           *Area
	AutoComplete   *util.AutoComplete
	Client         common.Client
	CommandHistory *CommandHistory
	Name           string
	IsAdmin        bool
	EnteredWorld   bool
	Created        bool // false = un-manifested "ghost" still in character creation
}

func (p *Player) Broadcast(msg string) {
	if !p.Client.SupportsTags() {
		if util.IsStateMessage(msg) || util.IsIdentityMessage(msg) || util.IsEventMessage(msg) {
			return
		}
		msg = util.StripTag(msg)
	}
	p.Client.SendMessage(msg)
}

func (p *Player) BroadcastState(w WorldLike, entityID common.EntityID) {
	health, err := w.GetComponent(entityID, "Health")
	if err != nil {
		return
	}
	h := health.(*Health)

	experience, _ := w.GetComponent(entityID, "Experience")
	level := 1
	currentXP := 0
	requiredXP := 100
	if experience != nil {
		exp := experience.(*Experience)
		exp.RLock()
		level = exp.Level
		currentXP = exp.Current
		requiredXP = CalculateRequiredXP(level)
		exp.RUnlock()
	}

	statusEffects, _ := w.GetComponent(entityID, "StatusEffects")
	hpBonus := 0
	var effectNames []string
	var fx []EffectView
	if statusEffects != nil {
		se := statusEffects.(*StatusEffects)
		hpBonus = se.GetTotalHPBonus()
		fx = se.Snapshot()
		se.RLock()
		for _, effect := range se.Effects {
			effectNames = append(effectNames, effect.Name)
		}
		se.RUnlock()
	}

	// Constitution raises the effective HP ceiling (same path as status bonuses),
	// and the raw stats ride along so the client can show them.
	var statsMap map[string]int
	var raceName string
	if statsComp, err := w.GetComponent(entityID, "Stats"); err == nil {
		if st, ok := statsComp.(*Stats); ok {
			hpBonus += st.HPBonus()
			raceName = st.Race
			statsMap = make(map[string]int, numStats)
			for _, t := range AllStats() {
				statsMap[StatAbbrev(t)] = st.Get(t)
			}
		}
	}

	// Worn gear raises the HP ceiling alongside stats and blessings.
	if eqComp, err := w.GetComponent(entityID, "Equipment"); err == nil {
		if eq, ok := eqComp.(*Equipment); ok {
			hpBonus += eq.HPBonus()
		}
	}

	// A beast form shows up as an effect tag (e.g. "Bear Form").
	if shComp, err := w.GetComponent(entityID, "Shift"); err == nil {
		if sh, ok := shComp.(*Shift); ok && sh.Form != "" {
			if form, ok := FormFor(sh.Form); ok {
				effectNames = append(effectNames, form.Label+" Form")
			}
		}
	}

	ep, maxEP := 0, 0
	if endComp, err := w.GetComponent(entityID, "Endurance"); err == nil {
		if end, ok := endComp.(*Endurance); ok {
			end.Regen()
			ep, maxEP = end.Current, end.Max
		}
	}

	areaTitle := "Unknown"
	if p.Area != nil {
		areaTitle = strings.TrimSpace(strings.SplitN(p.Area.Description, "\n", 2)[0])
	}

	var mountName string
	if mountComp, err := w.GetComponent(entityID, "Mount"); err == nil {
		if m, ok := mountComp.(*Mount); ok {
			mountName = m.Name
		}
	}

	gold := 0
	if invComp, err := w.GetComponent(entityID, "Inventory"); err == nil {
		if inv, ok := invComp.(*Inventory); ok {
			gold = inv.CountItem("gold_coin")
		}
	}

	// Authoritative status push for event-aware clients (replaces STATE|).
	vitals := charVitals{
		Type: "char.vitals", HP: h.Current, MaxHP: h.Max + hpBonus, EP: ep, MaxEP: maxEP, Level: level,
		XP: currentXP, ReqXP: requiredXP, Area: areaTitle, Gold: gold, Effects: effectNames, Fx: fx, Mount: mountName,
		Stats: statsMap, Race: raceName,
	}
	if data, err := json.Marshal(vitals); err == nil {
		p.Broadcast(util.TagMessage("EVENT", string(data)))
	}
}

func (p *Player) Look(w WorldLike) {
	p.Broadcast(p.DescribeArea(w))
	p.BroadcastRoomInfo()
}

// roomInfoEvent is the structured room header: name, exits, region.
type roomInfoEvent struct {
	Type   string   `json:"type"`
	Name   string   `json:"name"`
	Exits  []string `json:"exits"`
	Region string   `json:"region,omitempty"`
}

// BroadcastRoomInfo pushes the current room's name/exits/region so event-aware
// clients can drive the map and exit list authoritatively (not by scraping text).
func (p *Player) BroadcastRoomInfo() {
	if p.Area == nil {
		return
	}
	ev := roomInfoEvent{
		Type:   "room.info",
		Name:   strings.TrimSpace(strings.SplitN(p.Area.Description, "\n", 2)[0]),
		Region: p.Area.Region,
	}
	for _, e := range p.Area.Exits {
		ev.Exits = append(ev.Exits, e.Direction)
	}
	if data, err := json.Marshal(ev); err == nil {
		p.Broadcast(util.TagMessage("EVENT", string(data)))
	}
}

// DescribeArea returns information about the player's current area, including
// other players, NPCs, and exits.
func (p *Player) DescribeArea(w WorldLike) string {
	if p.Area == nil {
		return "You are nowhere."
	}

	var b strings.Builder

	b.WriteString(strings.TrimSpace(p.Area.Description))

	p.Area.PlayersMutex.RLock()
	var otherPlayers []string
	for _, player := range p.Area.Players {
		if player != p {
			otherPlayers = append(otherPlayers, player.Name)
		}
	}
	p.Area.PlayersMutex.RUnlock()

	npcs := p.Area.GetNPCs(w)
	corpses := p.Area.GetCorpses(w)
	items := p.Area.GetItems()

	hasEntities := len(otherPlayers) > 0 || len(npcs) > 0 || len(corpses) > 0 || len(items) > 0
	if hasEntities {
		b.WriteString("\n\n")

		hasCharacters := len(otherPlayers) > 0 || len(npcs) > 0

		for _, name := range otherPlayers {
			b.WriteString(name)
			b.WriteString(" is here.\n")
		}

		for _, npc := range npcs {
			b.WriteString(npc.Name)
			b.WriteString(" is here.\n")
		}

		hasCorpses := len(corpses) > 0

		if hasCorpses {
			if hasCharacters {
				b.WriteString("\n")
			}
			for _, corpse := range corpses {
				b.WriteString(corpse.GetDescription())
				b.WriteString(" is here.\n")
			}
		}

		if len(items) > 0 {
			if hasCharacters || hasCorpses {
				b.WriteString("\n")
			}
			for _, item := range items {
				if item.Stackable && item.Quantity > 1 {
					b.WriteString(fmt.Sprintf("%s x%d is here.\n", item.Name, item.Quantity))
				} else {
					b.WriteString(item.Name)
					b.WriteString(" is here.\n")
				}
			}
		}
	}

	if len(p.Area.Exits) > 0 {
		exits := make([]string, len(p.Area.Exits))
		for i, exit := range p.Area.Exits {
			exits[i] = exit.Direction
		}
		if hasEntities {
			b.WriteString("\n")
		} else {
			b.WriteString("\n\n")
		}
		b.WriteString("Exits: [")
		b.WriteString(strings.Join(exits, ", "))
		b.WriteString("]\n")
	}

	return b.String()
}
