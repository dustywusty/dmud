package game

import (
	"dmud/internal/components"
	"dmud/internal/util"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// commsFrame builds an EVENT|{comms} frame for event-aware clients (dropped for
// plain-text clients, who still get the legacy "X says:" line). The "self" flag
// marks the speaker's own copy so the client can render "You say:" etc.
func commsFrame(channel, from, to string, self bool, text string) string {
	data, _ := json.Marshal(struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
		From    string `json:"from,omitempty"`
		To      string `json:"to,omitempty"`
		Self    bool   `json:"self,omitempty"`
		Text    string `json:"text"`
	}{"comms", channel, from, to, self, text})
	return util.TagMessage("EVENT", string(data))
}

// sendChat delivers a chat line to one player: the structured event to a
// tag-capable client, the plain text otherwise.
func sendChat(p *components.Player, event, plain string) {
	if p.Client != nil && p.Client.SupportsTags() {
		p.Broadcast(event)
	} else {
		p.Broadcast(plain)
	}
}

func handleSay(player *components.Player, args []string, game *Game) {
	msg := strings.Join(args, " ")
	if msg == "" {
		player.Broadcast("Say what?")
		return
	}
	sendChat(player, commsFrame("say", "", "", true, msg), "You say: "+msg)
	player.Area.BroadcastChat(commsFrame("say", player.Name, "", false, msg),
		fmt.Sprintf("%s says: %s", player.Name, msg), player)

	// Check if this triggers NPC keyword responses
	game.handleSayToNPC(player, msg)
}

func handleTell(player *components.Player, args []string, game *Game) {
	if len(args) < 2 {
		player.Broadcast("Usage: tell <player> <message>")
		return
	}
	target, _ := game.findOnlinePlayer(strings.ToLower(args[0]))
	if target == nil {
		player.Broadcast("No such player online.")
		return
	}
	if target == player {
		player.Broadcast("Talking to yourself again?")
		return
	}
	msg := strings.Join(args[1:], " ")
	sendChat(target, commsFrame("tell", player.Name, "", false, msg),
		fmt.Sprintf("%s tells you: %s", player.Name, msg))
	sendChat(player, commsFrame("tell", "", target.Name, true, msg),
		fmt.Sprintf("You tell %s: %s", target.Name, msg))
}

func handleShout(player *components.Player, args []string, game *Game) {
	msg := strings.Join(args, " ")
	if msg == "" {
		player.Broadcast("Shout what?")
		return
	}
	game.HandleShout(player, msg)
}

func (g *Game) HandleShout(player *components.Player, msg string) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	if player.Area == nil {
		player.Broadcast("You shout but there is no sound.")
		return
	}
	log.Info().Msgf("Shout: %s", msg)

	depth := 10

	visited := make(map[*components.Area]bool)
	queue := []*components.Area{player.Area}

	for depth > 0 && len(queue) > 0 {
		depth--
		nextQueue := []*components.Area{}

		for _, area := range queue {
			visited[area] = true
			for _, exit := range area.Exits {
				if !visited[exit.Area] {
					visited[exit.Area] = true
					nextQueue = append(nextQueue, exit.Area)
				}
			}
		}
		queue = nextQueue
	}

	sendChat(player, commsFrame("shout", "", "", true, msg), "You shout: "+msg)
	for area := range visited {
		area.BroadcastChat(commsFrame("shout", player.Name, "", false, msg),
			fmt.Sprintf("%s shouts: %s", player.Name, msg), player)
	}
}
