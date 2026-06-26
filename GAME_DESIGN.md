# Game Design

The **why** behind the systems. This is hand-written design intent — it does
**not** restate numbers. For the live catalog (exact spells, races, costs, stat
mods) see [`docs/`](docs/README.md), which is generated from the code so it can't
drift.

## Sources of truth

One owner per thing:

| Content | Lives in | Edited by |
|---|---|---|
| Spells, classes, races, consumables, mounts, vendors, factions | Go registries (`internal/...`) | code |
| NPCs, spawns, areas, items | JSON in `resources/` | data |
| The human-readable catalog | `docs/*.md` | **generated** (`make docs`) |
| Design intent (this file) | `GAME_DESIGN.md` | hand-written |

A golden test (`TestDocsAreCurrent`) fails if `docs/` drifts from the registries,
so the catalog is always true. Never hand-edit `docs/`.

## Pillars

- **Earn it by doing it.** Progression is use-based: the actions you perform are
  the actions that improve. No grind screens — play the game you want to be good
  at and you get good at it.
- **Every action has a stat, every stat bends an action.** A clean loop: act →
  maybe train the governing stat → that stat makes the action better. Legible and
  self-reinforcing.
- **Resources gate spam, not fun.** Endurance stops infinite heals/nukes without
  cooldowns or mana micromanagement; it regenerates, and food/potions top it up.

## Onboarding (the ghost)

A brand-new character arrives as a **formless spirit** — un-manifested, and gated
to only the creation choices (and harmless info commands) until they've chosen a
**name**, a **race**, and a **class**. Completing all three **manifests** them
into the world, applies their class affinity, and (on a real client) wires up a
starter macro bar for their class spells. The `Created` flag persists, so a ghost
who logs out mid-creation resumes where they left off; established saves predating
the flag are auto-grandfathered as created. Rationale: the first thing a new
player does should be *become someone*, and they shouldn't be loose in the world
half-formed before that choice.

## Progression

Two axes that compose:

- **Level** (from kills) — gross power: bigger HP pool, unlocks higher-tier
  spells, scales melee/spell damage.
- **Stats** (from use) — fine, lateral growth. Each rolls a chance to rise per
  use, and the chance falls to zero at the cap, so the first points are fast and
  mastery is a long tail. A racial penalty trains *back* a little faster than a
  bonus, so races flavor a character without dooming it.

Stat → role: STR melee, INT arcane, WIS healing, CON toughness (HP **and**
endurance — a brawny ogre carries a deep stamina pool), DEX efficiency (cheaper
actions). The exact curves live in `internal/components/stats.go`.

## Classes & schools

Spells belong to **schools** (the real grouping). A **class** is a curated bundle
of schools plus a primary stat — the fantasy, not a hard wall (yet). Today classes
are descriptive; the natural next step is letting a player *pick* one, which then
gates which schools they can learn and grants an affinity bonus to the primary
stat. Designing class identity here keeps that future change a data edit, not a
rewrite.

**Balance intent:** offense (pyromancy) costs more endurance per point of effect
than utility/healing, because raw damage ends fights fastest. Control (domination)
is cheap but situational and can fail. Healing scales hardest with its stat (WIS)
to reward commitment to a support role.

## Races & scale

Races are stat presets plus a **size**. Size is the seed of "scale": today it
gates movement through narrow exits (a Large ogre can't follow you into a crawl);
later it can touch carry capacity, stealth, reach, and mounts. Keep size meaningful
but never a pure tax — a big body should trade agility/efficiency for raw power, as
the ogre does (huge STR/CON, dismal INT/DEX).

## NPCs

NPCs run on the **same** stat/race machinery as players, so a race is a creature's
whole identity in one word: an "ogre" NPC is automatically tanky (CON→HP) and
hits hard (STR→damage) and casts feebly (INT). New monster archetypes should be a
race + a template, not bespoke code.
