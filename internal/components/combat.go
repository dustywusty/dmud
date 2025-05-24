package components

import (
	"dmud/internal/common"
	"fmt"
	"sync"
)

type Combat struct {
	sync.RWMutex

	TargetID  common.EntityID
	MinDamage int
	MaxDamage int
}

func handleTargetDeath(w WorldLike, attackerID common.EntityID, targetID common.EntityID, attackerPlayer, targetPlayer *components.Player) {
    combat := &components.Combat{}
    combat.TargetID = ""

    w.RemoveComponent(attackerID, "Combat")
    w.RemoveComponent(targetID, "Combat")

    // Check if target is an NPC
    if targetPlayer == nil {
        if npcComponent, err := w.GetComponent(targetID, "NPC"); err == nil {
            npc := npcComponent.(*components.NPC)

            // Announce death
            if npc.Room != nil {
                npc.Room.Broadcast(npc.Name + " has been slain!")
            }

            // Remove NPC from world (spawn system will respawn it)
            w.RemoveEntity(targetID)

            // Award experience or loot here in the future
            attackerPlayer.Broadcast("You have defeated " + npc.Name + "!")
        }
    } else {
        // Player death
        attackerPlayer.Broadcast(fmt.Sprintf("You killed %s!", targetPlayer.Name))
        targetPlayer.Broadcast("You have died!")
    }
}