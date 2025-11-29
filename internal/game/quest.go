package game

import (
	"dmud/internal/components"
	"fmt"
	"strings"
)

func (g *Game) handleHail(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Hail whom? Usage: hail <npc>")
		return
	}

	targetName := strings.ToLower(strings.Join(args, " "))

	// Find NPCs in the area
	npcs := player.Area.GetNPCs(g.world.AsWorldLike())
	for _, npc := range npcs {
		if strings.Contains(strings.ToLower(npc.Name), targetName) {
			g.handleNPCHail(player, npc)
			return
		}
	}

	player.Broadcast("You don't see that NPC here.")
}

func (g *Game) handleNPCHail(player *components.Player, npc *components.NPC) {
	// Check if this NPC offers any quests with dialogue
	for _, quest := range components.QuestRegistry {
		if quest.NPCID == npc.TemplateID && quest.Dialogue != nil {
			player.Broadcast(fmt.Sprintf("%s says: %s", npc.Name, quest.Dialogue.Greeting))
			return
		}
	}

	// Default hail response for NPCs without quest dialogue
	if len(npc.Dialogue) > 0 {
		player.Broadcast(fmt.Sprintf("%s says: %s", npc.Name, npc.Dialogue[0]))
	} else {
		player.Broadcast(fmt.Sprintf("%s nods at you.", npc.Name))
	}
}

func (g *Game) handleSayToNPC(player *components.Player, keyword string) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))

	// Get player entity
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}

	// Use the dialogue handler from components
	handler := &components.QuestDialogueHandler{
		World: g.world.AsWorldLike(),
	}

	handler.HandleKeyword(player, keyword, playerEntity)
}
