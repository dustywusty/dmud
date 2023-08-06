package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
	"math/rand"
	"time"
)

type CombatSystem struct{}

func (cs *CombatSystem) Update(w *ecs.World, deltaTime float64) {
	rand.Seed(time.Now().UnixNano())

	attackingEntities, err := w.FindEntitiesByComponentPredicate("AttackingComponent", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, attackingEntity := range attackingEntities {
		attackingComponent, err := w.GetComponent(attackingEntity.ID, "AttackingComponent")
		if err != nil {
			continue
		}

		attacker := attackingComponent.(*components.AttackingComponent)

		if attacker.TargetID == "" {
			continue
		}

		targetHealthComponent, err := w.GetComponent(attacker.TargetID, "HealthComponent")
		if err != nil {
			continue
		}

		target := targetHealthComponent.(*components.HealthComponent)

		damage := rand.Intn(attacker.MaxDamage-attacker.MinDamage+1) + attacker.MinDamage

		target.CurrentHealth -= damage

		// Send message to attacker
		attackerPlayerComponent, err := w.GetComponent(attackingEntity.ID, "PlayerComponent")
		if err != nil {
			// Handle error
			continue
		}

		attackerPlayer := attackerPlayerComponent.(*components.PlayerComponent)

		attackerPlayer.Client.SendMessage(fmt.Sprintf("You attacked %s for %d damage!", attacker.TargetID, damage))
	}
}
