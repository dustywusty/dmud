package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/systems"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// damageSpellSpec parameterizes a single-target damage spell. The scaling stat
// (INT for pyromancy, WIS for lunar) both amplifies the spell and is trained by
// casting it.
type damageSpellSpec struct {
	minDmg, maxDmg int
	scaleStat      components.StatType
	lifesteal      float64 // fraction of damage healed back to the caster (0 = none)
	hitLine        string  // fmt args: targetName, damage
	areaVerb       string  // fmt args: casterName, targetName
	burnHP         int     // if >0, leaves a Burning DoT dealing this much HP per tick
	burnTicks      int     // number of burn ticks (with burnHP > 0)
}

// makeDamageSpell builds a SpellHandler for a damage spell from its spec.
func makeDamageSpell(spec damageSpellSpec) SpellHandler {
	return func(caster *components.Player, args []string, game *Game) {
		game.castDamageSpell(caster, args, spec)
	}
}

// castDamageSpell resolves a target NPC in the caster's room, rolls level- and
// stat-scaled damage, and applies it through the shared combat path (so the enemy
// HP bar, death, XP, and corpse all behave like melee).
func (g *Game) castDamageSpell(caster *components.Player, args []string, spec damageSpellSpec) {
	if caster.Area == nil {
		caster.Broadcast("You are nowhere.")
		return
	}

	casterID, err := g.getPlayerEntity(caster)
	if err != nil {
		caster.Broadcast("Something went wrong.")
		return
	}

	var targetNPC *components.NPC
	var targetID common.EntityID
	if len(args) == 0 {
		// No explicit target: default to whatever you're currently fighting, so a
		// one-press `cast <spell>` macro just works mid-combat.
		if combat, e := ecs.GetTypedComponent[*components.Combat](g.world, casterID, "Combat"); e == nil && combat != nil && combat.TargetID != "" {
			if npc, e2 := ecs.GetTypedComponent[*components.NPC](g.world, combat.TargetID, "NPC"); e2 == nil && npc != nil {
				targetNPC, targetID = npc, combat.TargetID
			}
		}
		if targetNPC == nil {
			caster.Broadcast("Cast it at whom? (you have no current target)")
			return
		}
	} else {
		targetNPC, targetID, err = g.findNPCInAreaByName(caster.Area, strings.Join(args, " "))
		if err != nil || targetNPC == nil {
			caster.Broadcast("You don't see that target here.")
			return
		}
	}

	damage := spec.minDmg + rand.Intn(spec.maxDmg-spec.minDmg+1)
	if exp, e := ecs.GetTypedComponent[*components.Experience](g.world, casterID, "Experience"); e == nil && exp != nil {
		damage = int(float64(damage) * components.GetLevelScaling(exp.GetLevel()))
	}
	stats := g.getStats(casterID)
	if stats != nil {
		damage = int(float64(damage) * stats.Factor(spec.scaleStat)) // the school's stat amplifies it
	}

	// ApplyPlayerSpellDamage handles the hit, the enemy HP bar, and (on a kill)
	// death/XP/corpse — exactly like a melee blow. NPC retaliation, if any, is the
	// AI system's job, same as for `kill`.
	dealt, _ := systems.ApplyPlayerSpellDamage(g.world, casterID, targetID, damage)

	caster.Broadcast(fmt.Sprintf(spec.hitLine, targetNPC.Name, dealt))
	caster.Area.Broadcast(fmt.Sprintf(spec.areaVerb, caster.Name, targetNPC.Name), caster)

	// Fire spells leave the target Burning (a DoT) if it survived the hit.
	if spec.burnHP > 0 && spec.burnTicks > 0 {
		if hp, e := ecs.GetTypedComponent[*components.Health](g.world, targetID, "Health"); e == nil && hp != nil && hp.Current > 0 {
			if se, e2 := g.getOrCreateStatusEffects(targetID); e2 == nil && se != nil {
				const burnInterval = 2 * time.Second
				se.AddEffect(components.StatusEffect{
					Type:           components.StatusEffectBurning,
					Name:           "Burning",
					AppliedAt:      time.Now(),
					Duration:       time.Duration(spec.burnTicks) * burnInterval,
					TickHP:         -spec.burnHP,
					TickInterval:   burnInterval,
					SourceEntityID: casterID,
				})
				caster.Broadcast(fmt.Sprintf("%s is wreathed in clinging flames.", targetNPC.Name))
			}
		}
	}

	// Life-drain spells heal the caster for a fraction of the damage dealt.
	if spec.lifesteal > 0 && dealt > 0 {
		if healed := int(float64(dealt) * spec.lifesteal); healed > 0 {
			if hp, e := ecs.GetTypedComponent[*components.Health](g.world, casterID, "Health"); e == nil && hp != nil {
				hp.Heal(healed)
				caster.Broadcast(fmt.Sprintf("You siphon %d life from %s.", healed, targetNPC.Name))
			}
		}
	}

	components.TrainStat(caster, stats, spec.scaleStat) // casting trains the school's stat
}

type SpellHandler func(caster *components.Player, args []string, game *Game)

type SpellDefinition struct {
	Name        string
	Aliases     []string
	Usage       string
	Description string
	School      string // thematic grouping; the seed of a future class label
	MinLevel    int    // caster level required to learn/cast; 0 or 1 = available immediately
	Cost        int    // endurance spent to cast; 0 = free
	Power       string // short human summary of effect (e.g. "10–18 fire dmg"); for docs
	Handler     SpellHandler
}

type npcControlSpellSpec struct {
	Usage                string
	EffectType           components.StatusEffectType
	EffectName           string
	DurationForLevel     func(level int) time.Duration
	CanTarget            func(template components.NPCTemplate) bool
	InvalidTargetMessage func(npcName string) string
	ConflictMessage      func(npcName string) string
	SuccessModifier      int
	ResistCasterMessage  func(npcName string) string
	ResistAreaMessage    func(casterName, npcName string) string
	CasterMessage        func(npcName string, duration time.Duration, renewed bool) string
	AreaMessage          func(casterName, npcName string, renewed bool) string
}

var spellRegistry = make(map[string]*SpellDefinition)
var maxSpellWords = 1

func init() {
	registerRestoration()
	registerSpell(&SpellDefinition{
		Name:        "control undead",
		Aliases:     []string{"controlundead", "command undead"},
		Usage:       "cast control undead <target>",
		Description: "Suppress an undead creature's aggression for a short time.",
		School:      "domination",
		Cost:        35,
		Power:       "subdue undead",
		Handler:     castControlUndead,
	})
	registerSpell(&SpellDefinition{
		Name:        "charm",
		Aliases:     []string{"charm person"},
		Usage:       "cast charm <target>",
		Description: "Beguile a living creature and suppress its aggression temporarily.",
		School:      "domination",
		Cost:        35,
		Power:       "beguile the living",
		Handler:     castCharm,
	})

	registerPyromancy()
	registerLunar()
	registerHoly()
	registerShadow()
}

// registerRestoration seeds the healing school: a ladder of heals, all scaled by
// the caster's Wisdom. Same shape as registerPyromancy ("multiply out").
func registerRestoration() {
	type healSpell struct {
		name, usage, desc string
		aliases           []string
		minLevel, cost    int
		base, perLevel    int
	}
	ladder := []healSpell{
		{"mend", "cast mend [target]", "A quick knitting of minor wounds.", nil, 1, 10, 8, 4},
		{"heal", "cast heal [target]", "Restore a solid amount of health to yourself or an ally.", []string{"healing"}, 1, 25, 15, 8},
		{"greater heal", "cast greater heal [target]", "Channel a powerful restorative surge.", []string{"greaterheal", "great heal"}, 8, 50, 40, 14},
	}
	for _, s := range ladder {
		registerSpell(&SpellDefinition{
			Name:        s.name,
			Aliases:     s.aliases,
			Usage:       s.usage,
			Description: s.desc,
			School:      "restoration", MinLevel: s.minLevel, Cost: s.cost,
			Power:   fmt.Sprintf("heal %d+%d/lvl (×WIS)", s.base, s.perLevel),
			Handler: makeHealSpell(healSpellSpec{name: s.name, base: s.base, perLevel: s.perLevel}),
		})
	}
}

// healSpellSpec parameterizes a restoration heal: amount = base + level*perLevel,
// then scaled by the caster's Wisdom.
type healSpellSpec struct {
	name     string
	base     int
	perLevel int
}

func makeHealSpell(spec healSpellSpec) SpellHandler {
	return func(caster *components.Player, args []string, game *Game) {
		game.castHealSpell(caster, args, spec)
	}
}

// castHealSpell heals the caster or a named ally, scaling the amount by the
// caster's Wisdom and clamping to the target's effective max (status + CON).
func (g *Game) castHealSpell(caster *components.Player, args []string, spec healSpellSpec) {
	target := caster
	if len(args) > 0 {
		name := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
		if name != "" && name != "me" && name != "self" {
			if caster.Area == nil {
				caster.Broadcast("You are nowhere.")
				return
			}
			candidate := caster.Area.GetPlayer(name)
			if candidate == nil {
				caster.Broadcast("You don't see that player here.")
				return
			}
			target = candidate
		}
	}

	targetEntityID, err := g.getPlayerEntity(target)
	if err != nil {
		caster.Broadcast("Unable to find that target.")
		return
	}

	level := 1
	if expComp, err := g.world.GetComponent(targetEntityID, "Experience"); err == nil {
		if exp, ok := expComp.(*components.Experience); ok {
			level = exp.GetLevel()
		}
	}

	amount := spec.base + level*spec.perLevel
	casterID, _ := g.getPlayerEntity(caster)
	casterStats := g.getStats(casterID)
	if casterStats != nil {
		amount = int(float64(amount) * casterStats.HealFactor()) // Wisdom amplifies healing
	}

	healthComp, err := g.world.GetComponent(targetEntityID, "Health")
	if err != nil {
		caster.Broadcast("Healing failed.")
		return
	}
	health := healthComp.(*components.Health)

	// Effective max includes status HP bonuses and the target's Constitution, so
	// healing tops off the HP the client actually shows.
	effectiveMax := health.Max
	if seComp, err := g.world.GetComponent(targetEntityID, "StatusEffects"); err == nil {
		if se, ok := seComp.(*components.StatusEffects); ok {
			effectiveMax += se.GetTotalHPBonus()
		}
	}
	if ts := g.getStats(targetEntityID); ts != nil {
		effectiveMax += ts.HPBonus()
	}

	missing := effectiveMax - health.Current
	if missing <= 0 {
		if target == caster {
			caster.Broadcast("You are already at full health.")
		} else {
			caster.Broadcast(fmt.Sprintf("%s is already at full health.", target.Name))
		}
		return
	}
	if amount > missing {
		amount = missing
	}
	health.Current += amount

	if target == caster {
		caster.Broadcast(fmt.Sprintf("You cast %s and restore %d health.", spec.name, amount))
	} else {
		caster.Broadcast(fmt.Sprintf("You cast %s on %s, restoring %d health.", spec.name, target.Name, amount))
		target.Broadcast(fmt.Sprintf("%s casts %s on you, restoring %d health.", caster.Name, spec.name, amount))
		if caster.Area != nil {
			caster.Area.Broadcast(fmt.Sprintf("%s casts %s on %s.", caster.Name, spec.name, target.Name), caster, target)
		}
	}

	components.TrainStat(caster, casterStats, components.WIS) // healing trains Wisdom
	target.BroadcastState(g.world.AsWorldLike(), targetEntityID)
}

// registerPyromancy seeds the fire pseudo-class: a single-target damage ladder
// gated by level and endurance. Adding another school later is just another
// table like this one ("multiply out").
func registerPyromancy() {
	type fireSpell struct {
		name        string
		aliases     []string
		minLevel    int
		cost        int
		minDmg      int
		maxDmg      int
		description string
		hitLine     string // fmt: target, damage
		areaVerb    string // fmt: caster, target
	}
	ladder := []fireSpell{
		{"spark", nil, 1, 8, 4, 8,
			"A flick of flame -- the first thing every pyromancer learns.",
			"You loose a spark at %s, singeing it for %d.",
			"%s flicks a spark at %s."},
		{"firebolt", []string{"fire bolt"}, 3, 16, 10, 18,
			"Hurl a bolt of fire at a single foe.",
			"You hurl a firebolt at %s, searing it for %d!",
			"%s hurls a firebolt at %s."},
		{"scorch", nil, 6, 24, 16, 28,
			"Wash a target in clinging flame.",
			"Flames wash over %s, scorching it for %d!",
			"%s sears %s with a gout of flame."},
		{"fireball", []string{"fire ball"}, 10, 40, 30, 50,
			"A roaring ball of fire that detonates on impact.",
			"A roaring fireball engulfs %s for %d!",
			"%s looses a roaring fireball at %s."},
		{"immolate", nil, 15, 60, 50, 80,
			"Wreathe a single enemy in white-hot, consuming fire.",
			"%s erupts in white-hot flame, taking %d!",
			"%s wreathes %s in consuming fire."},
	}
	for _, s := range ladder {
		spec := damageSpellSpec{minDmg: s.minDmg, maxDmg: s.maxDmg, scaleStat: components.INT, hitLine: s.hitLine, areaVerb: s.areaVerb}
		// Firebolt and up leave a lingering burn scaled to the spell's power.
		if s.minLevel >= 3 {
			spec.burnHP = s.minDmg / 5
			spec.burnTicks = 3
		}
		registerSpell(&SpellDefinition{
			Name:        s.name,
			Aliases:     s.aliases,
			Usage:       "cast " + s.name + " <target>",
			Description: s.description,
			School:      "pyromancy",
			MinLevel:    s.minLevel,
			Cost:        s.cost,
			// Power is built from the same numbers the handler uses, so it can't drift.
			Power:   fmt.Sprintf("%d–%d fire dmg (×INT)", s.minDmg, s.maxDmg),
			Handler: makeDamageSpell(spec),
		})
	}
}

// registerLunar seeds the Druid's moon-magic school: WIS-scaled radiant damage,
// the same ladder shape as pyromancy but powered by Wisdom rather than Intellect.
func registerLunar() {
	type moonSpell struct {
		name, desc        string
		minLevel, cost    int
		minDmg, maxDmg    int
		hitLine, areaVerb string
	}
	ladder := []moonSpell{
		{"moonfire", "A flickering mote of cold moonlight.", 1, 10, 5, 11,
			"Moonfire sears %s for %d.", "%s flicks cold moonfire at %s."},
		{"moonbeam", "A lancing beam of lunar radiance.", 6, 26, 18, 32,
			"A lunar beam scores %s for %d!", "%s calls down a beam of moonlight on %s."},
		{"starfall", "A rain of searing starlight.", 12, 55, 45, 75,
			"Starlight rains down on %s for %d!", "%s pulls a rain of stars down upon %s."},
	}
	for _, s := range ladder {
		spec := damageSpellSpec{minDmg: s.minDmg, maxDmg: s.maxDmg, scaleStat: components.WIS, hitLine: s.hitLine, areaVerb: s.areaVerb}
		registerSpell(&SpellDefinition{
			Name:        s.name,
			Usage:       "cast " + s.name + " <target>",
			Description: s.desc,
			School:      "lunar",
			MinLevel:    s.minLevel,
			Cost:        s.cost,
			Power:       fmt.Sprintf("%d–%d lunar dmg (×WIS)", s.minDmg, s.maxDmg),
			Handler:     makeDamageSpell(spec),
		})
	}
}

// registerHoly seeds the Cleric's battle-priest school: a WIS-scaled holy smite
// and a protective blessing.
func registerHoly() {
	registerSpell(&SpellDefinition{
		Name: "smite", Usage: "cast smite <target>",
		Description: "Call down a bolt of searing holy light.",
		School:      "holy", MinLevel: 3, Cost: 22,
		Power: "14–24 holy dmg (×WIS)",
		Handler: makeDamageSpell(damageSpellSpec{
			minDmg: 14, maxDmg: 24, scaleStat: components.WIS,
			hitLine: "Holy light scourges %s for %d!", areaVerb: "%s calls down holy light on %s.",
		}),
	})
	registerSpell(&SpellDefinition{
		Name: "bless", Usage: "cast bless [target]",
		Description: "Ward yourself or an ally with a protective blessing (+max HP).",
		School:      "holy", MinLevel: 5, Cost: 30,
		Power: "+40 max HP, 5m",
		Handler: makeBuffSpell(buffSpellSpec{
			name: "Blessed", effect: components.StatusEffectBlessed, hpBonus: 40, duration: 5 * time.Minute,
			selfMsg: "A holy ward wreathes %s (+%d max HP for %s).",
			areaMsg: "%s is wreathed in holy light.",
		}),
	})
}

// registerShadow seeds the Warlock's offensive school: a shadow bolt and a
// life-draining curse, both INT-scaled.
func registerShadow() {
	registerSpell(&SpellDefinition{
		Name: "shadow bolt", Aliases: []string{"shadowbolt"}, Usage: "cast shadow bolt <target>",
		Description: "Hurl a bolt of crackling shadow.",
		School:      "shadow", MinLevel: 1, Cost: 12,
		Power: "6–13 shadow dmg (×INT)",
		Handler: makeDamageSpell(damageSpellSpec{
			minDmg: 6, maxDmg: 13, scaleStat: components.INT,
			hitLine: "A bolt of shadow tears into %s for %d!", areaVerb: "%s hurls a bolt of shadow at %s.",
		}),
	})
	registerSpell(&SpellDefinition{
		Name: "drain", Usage: "cast drain <target>",
		Description: "Siphon an enemy's life into your own.",
		School:      "shadow", MinLevel: 5, Cost: 28,
		Power: "14–26 shadow dmg (×INT), heals 50%",
		Handler: makeDamageSpell(damageSpellSpec{
			minDmg: 14, maxDmg: 26, scaleStat: components.INT, lifesteal: 0.5,
			hitLine: "Your curse withers %s for %d!", areaVerb: "%s drains the life from %s.",
		}),
	})
}

// buffSpellSpec parameterizes a self/ally buff that applies a status effect.
type buffSpellSpec struct {
	name     string
	effect   components.StatusEffectType
	hpBonus  int
	duration time.Duration
	selfMsg  string // fmt args: targetName, hpBonus, duration
	areaMsg  string // fmt args: targetName
}

func makeBuffSpell(spec buffSpellSpec) SpellHandler {
	return func(caster *components.Player, args []string, game *Game) {
		game.castBuffSpell(caster, args, spec)
	}
}

// castBuffSpell applies a timed beneficial effect to the caster or a named ally.
func (g *Game) castBuffSpell(caster *components.Player, args []string, spec buffSpellSpec) {
	target := caster
	if len(args) > 0 {
		name := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
		if name != "" && name != "me" && name != "self" {
			if caster.Area == nil {
				caster.Broadcast("You are nowhere.")
				return
			}
			if cand := caster.Area.GetPlayer(name); cand != nil {
				target = cand
			} else {
				caster.Broadcast("You don't see that player here.")
				return
			}
		}
	}

	targetEntityID, err := g.getPlayerEntity(target)
	if err != nil {
		caster.Broadcast("Unable to find that target.")
		return
	}
	se, err := g.getOrCreateStatusEffects(targetEntityID)
	if err != nil {
		caster.Broadcast("The blessing fails to take hold.")
		return
	}
	se.AddEffect(components.StatusEffect{
		Type: spec.effect, Name: spec.name, AppliedAt: time.Now(),
		Duration: spec.duration, Applied: true, HPBonus: spec.hpBonus,
	})

	caster.Broadcast(fmt.Sprintf(spec.selfMsg, target.Name, spec.hpBonus, shortDuration(spec.duration)))
	if caster.Area != nil && spec.areaMsg != "" {
		caster.Area.Broadcast(fmt.Sprintf(spec.areaMsg, target.Name), caster)
	}
	if casterID, e := g.getPlayerEntity(caster); e == nil {
		components.TrainStat(caster, g.getStats(casterID), components.WIS)
	}
	target.BroadcastState(g.world.AsWorldLike(), targetEntityID)
}

func registerSpell(def *SpellDefinition) {
	if def == nil {
		return
	}

	keys := make([]string, 0, len(def.Aliases)+1)
	keys = append(keys, def.Name)
	keys = append(keys, def.Aliases...)

	for _, key := range keys {
		normalized := normalizeSpellKey(key)
		if normalized == "" {
			continue
		}
		spellRegistry[normalized] = def

		if words := len(strings.Fields(normalized)); words > maxSpellWords {
			maxSpellWords = words
		}
	}
}

func normalizeSpellKey(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return ""
	}
	return strings.Join(strings.Fields(normalized), " ")
}

func resolveSpellFromArgs(args []string) (*SpellDefinition, int) {
	if len(args) == 0 {
		return nil, 0
	}

	maxWords := maxSpellWords
	if len(args) < maxWords {
		maxWords = len(args)
	}

	for words := maxWords; words >= 1; words-- {
		candidate := normalizeSpellKey(strings.Join(args[:words], " "))
		if spell, ok := spellRegistry[candidate]; ok {
			return spell, words
		}
	}

	return nil, 1
}

// handleSpells prints the spellbook grouped by school, with each spell's level
// requirement, endurance cost, and whether the caster has unlocked it yet.
func (g *Game) handleSpells(player *components.Player, args []string, game *Game) {
	level := 1
	if casterID, err := g.getPlayerEntity(player); err == nil {
		level = g.casterLevel(casterID)
	}

	// Dedupe defs (the registry is keyed by name + every alias) and group by school.
	seen := make(map[*SpellDefinition]bool)
	bySchool := make(map[string][]*SpellDefinition)
	for _, def := range spellRegistry {
		if seen[def] {
			continue
		}
		seen[def] = true
		school := def.School
		if school == "" {
			school = "general"
		}
		bySchool[school] = append(bySchool[school], def)
	}

	schools := make([]string, 0, len(bySchool))
	for s := range bySchool {
		schools = append(schools, s)
	}
	sort.Strings(schools)

	player.Broadcast(fmt.Sprintf("== Spellbook == (you are level %d)", level))
	for _, school := range schools {
		defs := bySchool[school]
		sort.Slice(defs, func(i, j int) bool {
			if defs[i].MinLevel != defs[j].MinLevel {
				return defs[i].MinLevel < defs[j].MinLevel
			}
			return defs[i].Name < defs[j].Name
		})
		player.Broadcast(capitalize(school) + ":")
		for _, def := range defs {
			req := def.MinLevel
			if req < 1 {
				req = 1
			}
			status := ""
			if level < req {
				status = " (locked)"
			}
			player.Broadcast(fmt.Sprintf("  L%-2d %-14s %2d EP%s -- %s", req, def.Name, def.Cost, status, def.Description))
		}
	}
	player.Broadcast("Cast with: cast <spell> <target>")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func listKnownSpells() []string {
	unique := make(map[string]struct{})
	for _, spell := range spellRegistry {
		unique[spell.Name] = struct{}{}
	}

	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func castControlUndead(caster *components.Player, args []string, game *Game) {
	castNPCControlSpell(caster, args, game, npcControlSpellSpec{
		Usage:            "Usage: cast control undead <target>",
		EffectType:       components.StatusEffectControlledUndead,
		EffectName:       "Controlled Undead",
		DurationForLevel: controlUndeadDuration,
		CanTarget: func(template components.NPCTemplate) bool {
			return template.CreatureType == components.CreatureTypeUndead
		},
		InvalidTargetMessage: func(npcName string) string {
			return fmt.Sprintf("%s is not undead.", npcName)
		},
		ConflictMessage: func(npcName string) string {
			return fmt.Sprintf("%s is already controlled by another will.", npcName)
		},
		SuccessModifier: 10,
		ResistCasterMessage: func(npcName string) string {
			return fmt.Sprintf("%s resists your necromantic command.", npcName)
		},
		ResistAreaMessage: func(casterName, npcName string) string {
			return fmt.Sprintf("%s's command falters as %s resists.", casterName, npcName)
		},
		CasterMessage: func(npcName string, duration time.Duration, renewed bool) string {
			if renewed {
				return fmt.Sprintf("You renew your control over %s for %s.", npcName, shortDuration(duration))
			}
			return fmt.Sprintf("You seize control of %s for %s.", npcName, shortDuration(duration))
		},
		AreaMessage: func(casterName, npcName string, renewed bool) string {
			if renewed {
				return fmt.Sprintf("%s reasserts a necromantic command over %s.", casterName, npcName)
			}
			return fmt.Sprintf("%s utters a necromantic command and %s falls still.", casterName, npcName)
		},
	})
}

func castCharm(caster *components.Player, args []string, game *Game) {
	castNPCControlSpell(caster, args, game, npcControlSpellSpec{
		Usage:            "Usage: cast charm <target>",
		EffectType:       components.StatusEffectCharmed,
		EffectName:       "Charmed",
		DurationForLevel: charmDuration,
		CanTarget: func(template components.NPCTemplate) bool {
			return template.CreatureType != components.CreatureTypeUndead
		},
		InvalidTargetMessage: func(npcName string) string {
			return fmt.Sprintf("%s cannot be charmed by living magic.", npcName)
		},
		ConflictMessage: func(npcName string) string {
			return fmt.Sprintf("%s is already enthralled by another will.", npcName)
		},
		ResistCasterMessage: func(npcName string) string {
			return fmt.Sprintf("%s shrugs off your charm.", npcName)
		},
		ResistAreaMessage: func(casterName, npcName string) string {
			return fmt.Sprintf("%s's charm fizzles as %s resists.", casterName, npcName)
		},
		CasterMessage: func(npcName string, duration time.Duration, renewed bool) string {
			if renewed {
				return fmt.Sprintf("You deepen your charm over %s for %s.", npcName, shortDuration(duration))
			}
			return fmt.Sprintf("You charm %s for %s.", npcName, shortDuration(duration))
		},
		AreaMessage: func(casterName, npcName string, renewed bool) string {
			if renewed {
				return fmt.Sprintf("%s renews a beguiling charm over %s.", casterName, npcName)
			}
			return fmt.Sprintf("%s weaves beguiling magic around %s.", casterName, npcName)
		},
	})
}

func castNPCControlSpell(caster *components.Player, args []string, game *Game, spec npcControlSpellSpec) {
	if len(args) == 0 {
		usage := strings.TrimSpace(spec.Usage)
		if usage == "" {
			usage = "Usage: cast <spell> <target>"
		}
		caster.Broadcast(usage)
		return
	}
	if caster.Area == nil {
		caster.Broadcast("You are nowhere.")
		return
	}

	targetName := strings.TrimSpace(strings.Join(args, " "))
	npc, npcEntityID, err := game.findNPCInAreaByName(caster.Area, targetName)
	if err != nil {
		caster.Broadcast(fmt.Sprintf("Invalid target: %v", err))
		return
	}
	if npc == nil {
		caster.Broadcast("You don't see that here.")
		return
	}

	npcName := npc.Name
	npcTemplateID := npc.TemplateID

	template, ok := components.NPCTemplates[npcTemplateID]
	if !ok {
		caster.Broadcast("The spell cannot find purchase on that target.")
		return
	}
	if spec.CanTarget != nil && !spec.CanTarget(template) {
		if spec.InvalidTargetMessage != nil {
			caster.Broadcast(spec.InvalidTargetMessage(npcName))
		} else {
			caster.Broadcast(fmt.Sprintf("%s is not a valid target.", npcName))
		}
		return
	}

	casterEntityID, err := game.getPlayerEntity(caster)
	if err != nil {
		caster.Broadcast("You lose focus and the spell fizzles.")
		return
	}

	level := 1
	if expComp, err := game.world.GetComponent(casterEntityID, "Experience"); err == nil {
		if exp, ok := expComp.(*components.Experience); ok {
			level = exp.GetLevel()
		}
	}

	duration := 45 * time.Second
	if spec.DurationForLevel != nil {
		duration = spec.DurationForLevel(level)
	}

	statusEffects, err := game.getOrCreateStatusEffects(npcEntityID)
	if err != nil {
		caster.Broadcast("The magic fails to take hold.")
		return
	}

	if controlEffect, ok := statusEffects.GetActiveSuppressionEffect(); ok {
		if controlEffect.SourceEntityID != "" && controlEffect.SourceEntityID != casterEntityID {
			if spec.ConflictMessage != nil {
				caster.Broadcast(spec.ConflictMessage(npcName))
			} else {
				caster.Broadcast(fmt.Sprintf("%s is already under another controlling effect.", npcName))
			}
			return
		}
	}

	_, renewed := statusEffects.GetEffect(spec.EffectType)
	if !renewed {
		successChance := controlSpellSuccessChance(level, template, spec.SuccessModifier)
		if !rollPercent(successChance) {
			if spec.ResistCasterMessage != nil {
				caster.Broadcast(spec.ResistCasterMessage(npcName))
			} else {
				caster.Broadcast(fmt.Sprintf("%s resists your spell.", npcName))
			}
			if spec.ResistAreaMessage != nil {
				areaMessage := strings.TrimSpace(spec.ResistAreaMessage(caster.Name, npcName))
				if areaMessage != "" {
					caster.Area.Broadcast(areaMessage, caster)
				}
			}
			return
		}
	}

	statusEffects.AddEffect(components.StatusEffect{
		Type:                spec.EffectType,
		Name:                spec.EffectName,
		AppliedAt:           time.Now(),
		Duration:            duration,
		Applied:             true,
		SourceEntityID:      casterEntityID,
		SuppressAggro:       true,
		SuppressRetaliation: true,
	})

	if combat, err := ecs.GetTypedComponent[*components.Combat](game.world, npcEntityID, "Combat"); err == nil {
		combat.TargetID = ""
		combat.TargetQueue = nil
	}

	if spec.CasterMessage != nil {
		caster.Broadcast(spec.CasterMessage(npcName, duration, renewed))
	} else {
		caster.Broadcast(fmt.Sprintf("You control %s for %s.", npcName, shortDuration(duration)))
	}

	if spec.AreaMessage != nil {
		areaMessage := strings.TrimSpace(spec.AreaMessage(caster.Name, npcName, renewed))
		if areaMessage != "" {
			caster.Area.Broadcast(areaMessage, caster)
		}
	}
}

func controlUndeadDuration(level int) time.Duration {
	if level < 1 {
		level = 1
	}

	duration := 45*time.Second + (time.Duration(level) * 15 * time.Second)
	if duration > 5*time.Minute {
		return 5 * time.Minute
	}
	return duration
}

func charmDuration(level int) time.Duration {
	if level < 1 {
		level = 1
	}

	duration := 30*time.Second + (time.Duration(level) * 10 * time.Second)
	if duration > 3*time.Minute {
		return 3 * time.Minute
	}
	return duration
}

func controlSpellSuccessChance(level int, template components.NPCTemplate, successModifier int) int {
	if level < 1 {
		level = 1
	}

	targetPower := (template.Health / 20) + ((template.MinDamage + template.MaxDamage) / 6)
	chance := 60 + (level * 5) - (targetPower * 5) + successModifier

	if chance < 20 {
		return 20
	}
	if chance > 90 {
		return 90
	}
	return chance
}

func rollPercent(chance int) bool {
	if chance <= 0 {
		return false
	}
	if chance >= 100 {
		return true
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return rng.Intn(100) < chance
}

func shortDuration(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	leftoverSeconds := seconds % 60
	if leftoverSeconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	return fmt.Sprintf("%dm%ds", minutes, leftoverSeconds)
}

func (g *Game) findNPCInAreaByName(area *components.Area, rawName string) (*components.NPC, common.EntityID, error) {
	if area == nil {
		return nil, "", nil
	}

	matcher, _, _, err := buildItemMatcher(rawName)
	if err != nil {
		return nil, "", err
	}

	npcs := area.GetNPCs(g.world.AsWorldLike())
	for _, npc := range npcs {
		if !matcher(npc.Name) {
			continue
		}

		npcEntities, _ := g.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
			n, ok := i.(*components.NPC)
			return ok && n == npc
		})
		if len(npcEntities) == 0 {
			continue
		}

		return npc, npcEntities[0].ID, nil
	}

	return nil, "", nil
}

func (g *Game) getOrCreateStatusEffects(entityID common.EntityID) (*components.StatusEffects, error) {
	statusEffects, err := ecs.GetTypedComponent[*components.StatusEffects](g.world, entityID, "StatusEffects")
	if err == nil && statusEffects != nil {
		return statusEffects, nil
	}

	entity, err := g.world.FindEntity(entityID)
	if err != nil {
		return nil, err
	}

	statusEffects = components.NewStatusEffects()
	g.world.AddComponent(&entity, statusEffects)
	return statusEffects, nil
}
