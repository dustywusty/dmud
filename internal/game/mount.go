package game

import (
	"dmud/internal/components"
	"dmud/internal/systems"
	"fmt"
	"strings"
	"time"
)

// directionAliases maps movement shorthands and full names to a canonical
// direction, used to tell `ride <horse>` from `ride <direction>`.
var directionAliases = map[string]string{
	"n": "north", "north": "north",
	"s": "south", "south": "south",
	"e": "east", "east": "east",
	"w": "west", "west": "west",
	"u": "up", "up": "up",
	"d": "down", "down": "down",
}

func canonicalDirection(s string) (string, bool) {
	dir, ok := directionAliases[strings.ToLower(strings.TrimSpace(s))]
	return dir, ok
}

// handleRide is the umbrella verb: `ride <direction>` gallops, anything else is
// treated as `mount <horse>`.
func (g *Game) handleRide(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Ride what, or which way? Usage: 'ride <horse>' to mount, 'ride <direction>' to gallop.")
		return
	}
	if dir, ok := canonicalDirection(args[0]); ok {
		g.gallop(player, dir)
		return
	}
	g.handleMount(player, args, game)
}

// handleMount saddles a horse the player owns.
func (g *Game) handleMount(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}

	if _, err := g.world.GetComponent(entityID, "Mount"); err == nil {
		player.Broadcast("You're already in the saddle. 'dismount' first.")
		return
	}

	invComp, err := g.world.GetComponent(entityID, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory!")
		return
	}
	inventory := invComp.(*components.Inventory)

	wanted := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
	var stats *components.MountStats
	for _, item := range inventory.GetItems() {
		if !components.IsMountItem(item.ID) {
			continue
		}
		if wanted == "" || strings.Contains(strings.ToLower(item.Name), wanted) {
			stats, _ = components.MountStatsFor(item.ID)
			break
		}
	}

	if stats == nil {
		if wanted == "" {
			player.Broadcast("You don't own a horse. Buy one from a stablemaster (try 'list').")
		} else {
			player.Broadcast("You don't have that horse.")
		}
		return
	}

	mount := &components.Mount{
		ItemID:       stats.ItemID,
		Name:         stats.Name,
		Speed:        stats.Speed,
		Endurance:    stats.MaxEndurance,
		MaxEndurance: stats.MaxEndurance,
	}
	mount.Regen() // stamps LastRest

	entity, err := g.world.FindEntity(entityID)
	if err != nil {
		return
	}
	g.world.AddComponent(&entity, mount)

	player.Broadcast(fmt.Sprintf("You swing up into the saddle of your %s. (move speed %d, endurance %d) Use 'ride <direction>' to gallop.", mount.Name, mount.Speed, mount.MaxEndurance))
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s climbs into the saddle of a %s.", player.Name, mount.Name), player)
	}
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}

// handleDismount gets the player off their horse.
func (g *Game) handleDismount(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}

	comp, err := g.world.GetComponent(entityID, "Mount")
	if err != nil {
		player.Broadcast("You're not mounted.")
		return
	}
	mount := comp.(*components.Mount)

	g.world.RemoveComponent(entityID, "Mount")
	player.Broadcast(fmt.Sprintf("You hop down from your %s.", mount.Name))
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s dismounts.", player.Name), player)
	}
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}

// gallop moves a mounted player up to their horse's Speed in rooms in one
// command, consuming endurance for each room beyond the first. This is the
// concrete payoff of a horse's move speed and endurance.
func (g *Game) gallop(player *components.Player, dir string) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}

	comp, err := g.world.GetComponent(entityID, "Mount")
	if err != nil {
		player.Broadcast("You're on foot. Mount a horse first (try 'mount').")
		return
	}
	mount := comp.(*components.Mount)

	if healthComp, err := g.world.GetComponent(entityID, "Health"); err == nil {
		if health, ok := healthComp.(*components.Health); ok && health.Status == components.Dead {
			player.Broadcast("You are dead.")
			return
		}
	}

	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	mount.Regen()

	casterID, idErr := g.getPlayerEntity(player)

	rooms := 0
	consumed := false
	for i := 0; i < mount.Speed; i++ {
		if i > 0 && mount.Endurance <= 0 {
			player.Broadcast(fmt.Sprintf("Your %s is winded and slows to a walk.", mount.Name))
			break
		}

		exit := player.Area.GetExit(dir)
		if exit == nil {
			if rooms == 0 {
				player.Broadcast("You can't ride that way.")
			} else {
				player.Broadcast("The trail ends here.")
			}
			break
		}

		from := player.Area
		player.Area.RemovePlayer(player)
		player.Area = exit.Area
		player.Area.AddPlayer(player)
		if idErr == nil {
			systems.MoveControlledFollowers(g.world, player, casterID, from, exit.Area)
		}
		rooms++

		if i > 0 {
			mount.Endurance--
			consumed = true
		}
	}

	if rooms == 0 {
		return
	}

	if consumed {
		mount.LastRest = time.Now() // start the rest clock from this gallop
	}

	player.Broadcast(fmt.Sprintf("You ride %s on your %s.", dir, mount.Name))
	player.Look(g.world.AsWorldLike())
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}
