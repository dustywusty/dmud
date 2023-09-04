package game

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/systems"
	"dmud/internal/util"

	"github.com/jedib0t/go-pretty/table"
	"github.com/rs/zerolog/log"
)

///////////////////////////////////////////////////////////////////////////////////////////////
// Game
//

type Game struct {
	defaultRoom *components.RoomComponent

	players   map[string]*ecs.Entity
	playersMu sync.Mutex

	world *ecs.World

	AddPlayerChan    chan common.Client
	RemovePlayerChan chan common.Client
	CommandChan      chan ClientCommand
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (g *Game) HandleConnect(c common.Client) {
	playerComponent := &components.PlayerComponent{
		Client: c,
		Name:   util.GenerateRandomName(),
		Room:   g.defaultRoom,
	}
	healthComponent := &components.HealthComponent{
		MaxHealth:     100,
		CurrentHealth: 100,
	}

	g.Broadcast(fmt.Sprintf("%s has joined the game.", playerComponent.Name), c)

	g.playersMu.Lock()

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)

	g.world.AddComponent(&playerEntity, playerComponent)
	g.world.AddComponent(&playerEntity, healthComponent)

	g.players[playerComponent.Name] = &playerEntity
	g.defaultRoom.AddPlayer(playerComponent)

	g.playersMu.Unlock()

	c.SendMessage(util.WelcomeBanner)
	c.SendMessage(g.defaultRoom.Description)

	go c.HandleRequest()
}

func (g *Game) HandleDisconnect(c common.Client) {
	g.playersMu.Lock()

	player, err := g.getPlayer(c)
	if err != nil {
		log.Error().Err(err).Msg("Error getting disconnected player")
		g.playersMu.Unlock()
		return
	}

	if playerEntity, ok := g.players[player.Name]; ok {
		g.world.RemoveEntity(playerEntity.ID)
	}

	log.Info().Msgf("Player count before disconnect: %d", len(g.players))
	delete(g.players, player.Name)
	log.Info().Msgf("Player count after disconnect: %d", len(g.players))

	g.playersMu.Unlock()

	if err := c.CloseConnection(); err != nil {
		log.Error().Err(err).Msg("Error closing client connection")
		return
	}

	g.Broadcast(fmt.Sprintf("%s has left the game.", player.Name), c)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (g *Game) getPlayer(c common.Client) (*components.PlayerComponent, error) {
	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "PlayerComponent")
		if err != nil {
			return nil, fmt.Errorf("error getting player component for entity id %s, %v", playerEntity.ID, err)
		}

		player, ok := playerComponent.(*components.PlayerComponent)
		if !ok {
			return nil, fmt.Errorf("unable to cast component to PlayerComponent")
		}

		if player.Client == c {
			return player, nil
		}
	}
	return nil, fmt.Errorf("player not found")
}

func (g *Game) handleCommand(c ClientCommand) {
	command := c.Command
	client := c.Client

	player, err := g.getPlayer(client)
	if err != nil {
		log.Warn().Msg(fmt.Sprintf("Error getting player component: %s", err))
		return
	}

	switch c.Command.Cmd {
	case "exit":
		g.handleExit(player, command)
	case "k", "kill":
		g.handleKill(player, command)
	case "look":
		player.Look()
	case "n", "s", "e", "w", "u", "d", "north", "south", "east", "west", "up", "down":
		dirMapping := map[string]string{
			"n": "north",
			"s": "south",
			"e": "east",
			"w": "west",
			"u": "up",
			"d": "down",
		}
		fullDir := c.Command.Cmd
		if shortDir, ok := dirMapping[c.Command.Cmd]; ok {
			fullDir = shortDir
		}
		player.Move(fullDir)
	case "name":
		g.handleRename(player, command)
	case "scan":
		g.handleScan(player, command)
	case "say":
		g.handleSay(player, command)
	case "shout":
		g.handleSay(player, command)
	case "who":
		g.handleWho(player, command)
	default:
		client.SendMessage(fmt.Sprintf("What do you mean, \"%s\"?", command.Cmd))
	}
}

func (g *Game) handleExit(player *components.PlayerComponent, command Command) {
	g.RemovePlayerChan <- player.Client
	g.Broadcast(fmt.Sprintf("%s has left the game.", player.Name), player.Client)
}

func (g *Game) handleRename(player *components.PlayerComponent, command Command) {
	if (len(command.Args) == 0) || (len(command.Args) > 1) {
		player.Broadcast(player.Name)
		return
	}

	newName := command.Args[0]
	oldName := player.Name

	player.Lock()
	player.Name = newName
	g.players[newName] = g.players[oldName]
	delete(g.players, oldName)
	player.Unlock()

	g.Broadcast(fmt.Sprintf("%s has changed their name to %s", oldName, player.Name), player.Client)
}

func (g *Game) handleSay(player *components.PlayerComponent, command Command) {
	msg := strings.Join(command.Args, " ")
	if msg == "" {
		player.Broadcast("Say what?")
		return
	}
	// player.Broadcast(fmt.Sprintf("You say: %s", msg))
	player.Room.Broadcast(fmt.Sprintf("%s says: %s", player.Name, msg))
}

func (g *Game) handleScan(player *components.PlayerComponent, command Command) {
	exits := []string{}
	for _, exit := range player.Room.Exits {
		exits = append(exits, exit.Direction)
	}
	player.Broadcast("Exits: " + strings.Join(exits, ", "))
}

func (g *Game) handleShout(p *components.PlayerComponent, msg string, depths ...int) {
	if p.Room == nil {
		p.Broadcast("You shout but there is no sound")
		return
	}
	log.Info().Msgf("Shout: %s", msg)

	depth := 10
	if len(depths) > 0 {
		depth = depths[0]
	}

	visited := make(map[*components.RoomComponent]bool)
	queue := []*components.RoomComponent{p.Room}

	for depth > 0 && len(queue) > 0 {
		depth--
		nextQueue := []*components.RoomComponent{}

		for _, room := range queue {
			visited[room] = true
			for _, exit := range room.Exits {
				if !visited[exit.Room] {
					visited[exit.Room] = true
					nextQueue = append(nextQueue, exit.Room)
				}
			}
		}
		queue = nextQueue
	}

	for room := range visited {
		room.Broadcast(p.Name+" shouts: "+msg, p)
	}
}

func (g *Game) handleWho(player *components.PlayerComponent, command Command) {
	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"ID", "Name"})

	players := []string{}
	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "PlayerComponent")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get PlayerComponent for player %s", playerEntity.ID)
			continue
		}

		player, ok := playerComponent.(*components.PlayerComponent)
		if !ok {
			log.Error().Msgf("Error type asserting PlayerComponent for player %s", playerEntity.ID)
			continue
		}

		tw.AppendRow(table.Row{playerEntity.ID, player.Name})
	}

	player.Broadcast(tw.Render())
	log.Trace().Msgf("Players: %s", strings.Join(players, ", "))
}

func (g *Game) handleKill(player *components.PlayerComponent, command Command) {
	targetEntity := g.players[strings.Join(command.Args, " ")]
	if targetEntity == nil {
		player.Broadcast("Kill who?")
		return
	}

	playerEntity := g.players[player.Name]
	if playerEntity == nil {
		log.Warn().Msg(fmt.Sprintf("Error getting player's own entity for %s", player.Name))
		return
	}

	attackingPlayer := components.CombatComponent{
		TargetID:  targetEntity.ID,
		MinDamage: 1,
		MaxDamage: 5,
	}

	g.world.AddComponent(playerEntity, &attackingPlayer)
}

func (g *Game) loop() {
	updateTicker := time.NewTicker(10 * time.Millisecond)
	defer updateTicker.Stop()

	for {
		select {
		case client := <-g.AddPlayerChan:
			g.HandleConnect(client)
		case client := <-g.RemovePlayerChan:
			g.HandleDisconnect(client)
		case command := <-g.CommandChan:
			g.handleCommand(command)
		case <-updateTicker.C:
			g.world.Update()
		}
	}
}

func (g *Game) Broadcast(m string, excludeClients ...common.Client) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "PlayerComponent")
		if err != nil {
			log.Error().Msgf("error getting player component for entity id %s, %v", playerEntity.ID, err)
			continue
		}

		player, ok := playerComponent.(*components.PlayerComponent)
		if !ok {
			log.Error().Msgf("unable to cast component to PlayerComponent %v", playerComponent)
			continue
		}

		player.RWMutex.RLock()
		if !util.ContainsClient(excludeClients, player.Client) {
			player.Broadcast(m)
		}
		player.RWMutex.RUnlock()
	}

	log.Trace().Msgf("Broadcast: %s", m)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func NewGame() *Game {
	combatSytem := &systems.CombatSystem{}

	world := ecs.NewWorld()
	world.AddSystem(combatSytem)

	defaultRoom, _ := world.GetComponent("1", "RoomComponent")

	game := Game{
		defaultRoom:      defaultRoom.(*components.RoomComponent),
		players:          make(map[string]*ecs.Entity),
		world:            world,
		AddPlayerChan:    make(chan common.Client),
		RemovePlayerChan: make(chan common.Client),
		CommandChan:      make(chan ClientCommand),
	}

	go game.loop()

	return &game
}
