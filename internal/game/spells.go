package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

type SpellHandler func(caster *components.Player, args []string, game *Game)

type SpellDefinition struct {
	Name        string
	Aliases     []string
	Usage       string
	Description string
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
	registerSpell(&SpellDefinition{
		Name:        "heal",
		Aliases:     []string{"healing"},
		Usage:       "cast heal [target]",
		Description: "Restore health to yourself or another player.",
		Handler:     castHeal,
	})
	registerSpell(&SpellDefinition{
		Name:        "control undead",
		Aliases:     []string{"controlundead", "command undead"},
		Usage:       "cast control undead <target>",
		Description: "Suppress an undead creature's aggression for a short time.",
		Handler:     castControlUndead,
	})
	registerSpell(&SpellDefinition{
		Name:        "charm",
		Aliases:     []string{"charm person"},
		Usage:       "cast charm <target>",
		Description: "Beguile a living creature and suppress its aggression temporarily.",
		Handler:     castCharm,
	})
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

	npc.RLock()
	npcName := npc.Name
	npcTemplateID := npc.TemplateID
	npc.RUnlock()

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
