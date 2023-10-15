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

// -----------------------------------------------------------------------------

type Game struct {
	defaultRoom *components.Room

	players   map[string]*ecs.Entity
	playersMu sync.Mutex

	world *ecs.World

	AddPlayerChan    chan common.Client
	RemovePlayerChan chan common.Client
	CommandChan      chan ClientCommand
}

// -----------------------------------------------------------------------------

func (g *Game) HandleConnect(c common.Client) {
	playerComponent := &components.Player{
		Client: c,
		Name:   util.GenerateRandomName(),
		Room:   g.defaultRoom,
	}
	healthComponent := &components.Health{
		MaxHealth:     100,
		CurrentHealth: 100,
	}

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)

	g.world.AddComponent(&playerEntity, playerComponent)
	g.world.AddComponent(&playerEntity, healthComponent)

	g.playersMu.Lock()

	g.players[playerComponent.Name] = &playerEntity
	g.defaultRoom.AddPlayer(playerComponent)

	g.playersMu.Unlock()

	playerComponent.Broadcast(util.WelcomeBanner)
	playerComponent.Broadcast(g.defaultRoom.Description)

	g.Broadcast(fmt.Sprintf("%s has joined the game.", playerComponent.Name), c)

	go c.HandleRequest()
}

// -----------------------------------------------------------------------------

func (g *Game) HandleDisconnect(c common.Client) {
	player, err := g.getPlayer(c)
	if err != nil {
		return
	}

	g.playersMu.Lock()

	var playerEntity = g.players[player.Name]
	if playerEntity == nil {
		log.Error().Msg("Player entity was nil")
		return
	}

	g.world.RemoveEntity(playerEntity.ID)

	log.Trace().Msgf("Number of players: %d", len(g.players))
	delete(g.players, player.Name)
	log.Trace().Msgf("Number of players: %d", len(g.players))

	g.playersMu.Unlock()
	c.CloseConnection()

	g.Broadcast(fmt.Sprintf("%s has left the game.", player.Name), c)
}

// -----------------------------------------------------------------------------

func (g *Game) getPlayer(c common.Client) (*components.Player, error) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()
	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			return nil, fmt.Errorf("error getting player component for entity id %s, %v", playerEntity.ID, err)
		}

		player, ok := playerComponent.(*components.Player)
		if !ok {
			return nil, fmt.Errorf("unable to cast component to Player")
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
		g.handleMove(player, command)
	case "name":
		g.handleRename(player, command)
	case "say":
		g.handleSay(player, command)
	case "scan":
		g.handleScan(player, command)
	case "shout":
		g.handleShout(player, strings.Join(command.Args, " "))
	case "who":
		g.handleWho(player, command)
	default:
		player.Broadcast(fmt.Sprintf("What do you mean, \"%s\"?", command.Cmd))
	}
}

// -----------------------------------------------------------------------------

func (g *Game) handleExit(player *components.Player, command Command) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	g.HandleDisconnect(player.Client)
}

// -----------------------------------------------------------------------------

func (g *Game) handleLook(player *components.Player, command Command) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	player.Look()
}

// -----------------------------------------------------------------------------

func (g *Game) handleMove(player *components.Player, command Command) {
	dirMapping := map[string]string{
		"n": "north",
		"s": "south",
		"e": "east",
		"w": "west",
		"u": "up",
		"d": "down",
	}

	fullDir := command.Cmd
	if shortDir, ok := dirMapping[command.Cmd]; ok {
		fullDir = shortDir
	}

	player.RWMutex.RLock()
	playerEntity := g.players[player.Name]
	player.RWMutex.RUnlock()

	if playerEntity == nil {
		log.Warn().Msg(fmt.Sprintf("Error getting player's own entity for %s", player.Name))
		return
	}

	movement := components.Movement{
		Direction: fullDir,
		Status:    components.Walking,
	}
	g.world.AddComponent(playerEntity, &movement)
}

// -----------------------------------------------------------------------------

func (g *Game) handleRename(player *components.Player, command Command) {
	player.Lock()
	defer player.Unlock()

	if (len(command.Args) == 0) || (len(command.Args) > 1) {
		player.Broadcast(player.Name)
		return
	}

	newName := command.Args[0]
	oldName := player.Name

	player.Name = newName
	g.players[newName] = g.players[oldName]
	delete(g.players, oldName)

	g.Broadcast(fmt.Sprintf("%s has changed their name to %s", oldName, player.Name))
}

// -----------------------------------------------------------------------------

func (g *Game) handleSay(player *components.Player, command Command) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	msg := strings.Join(command.Args, " ")
	if msg == "" {
		player.Broadcast("Say what?")
		return
	}

	player.Room.Broadcast(fmt.Sprintf("%s says: %s", player.Name, msg))
}

// -----------------------------------------------------------------------------

func (g *Game) handleScan(player *components.Player, command Command) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	exits := []string{}
	for _, exit := range player.Room.Exits {
		exits = append(exits, exit.Direction)
	}

	player.Broadcast("Exits: " + strings.Join(exits, ", "))
}

// -----------------------------------------------------------------------------

func (g *Game) handleShout(player *components.Player, msg string, depths ...int) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	if player.Room == nil {
		player.Broadcast("You shout but there is no sound")
		return
	}
	log.Info().Msgf("Shout: %s", msg)

	depth := 10
	if len(depths) > 0 {
		depth = depths[0]
	}

	visited := make(map[*components.Room]bool)
	queue := []*components.Room{player.Room}

	for depth > 0 && len(queue) > 0 {
		depth--
		nextQueue := []*components.Room{}

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
		room.Broadcast(player.Name+" shouts: "+msg, player)
	}
}

// -----------------------------------------------------------------------------

func (g *Game) handleWho(player *components.Player, command Command) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	tw := table.NewWriter()
	tw.SetStyle(table.StyleLight)
	tw.AppendHeader(table.Row{"Player", "Race", "Class", "Online Since"})

	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get component for player %s", playerEntity.ID)
			continue
		}
		player, ok := playerComponent.(*components.Player)
		if !ok {
			log.Error().Msgf("Error type asserting component for player %s", playerEntity.ID)
			continue
		}
		tw.AppendRow(table.Row{player.Name, "??", "??", playerEntity.CreatedAt.DiffForHumans()})
	}

	player.Broadcast(tw.Render())
}

// -----------------------------------------------------------------------------

func (g *Game) handleKill(player *components.Player, command Command) {
	log.Trace().Msgf("Kill: %s", command.Args)

	targetEntity := g.players[strings.Join(command.Args, " ")]
	if targetEntity == nil {
		player.Broadcast("Kill who?")
		return
	}

	g.playersMu.Lock()
	playerEntity := g.players[player.Name]
	g.playersMu.Unlock()

	if playerEntity == nil {
		log.Warn().Msg(fmt.Sprintf("Error getting player's own entity for %s", player.Name))
		return
	}

	attackingPlayer := components.Combat{
		TargetID:  targetEntity.ID,
		MinDamage: 10,
		MaxDamage: 50,
	}

	g.world.AddComponent(playerEntity, &attackingPlayer)
}

// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------

func (g *Game) Broadcast(m string, excludeClients ...common.Client) {
	log.Info().Msgf("Broadcasting: %s", m)

	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			log.Error().Msgf("error getting player for entity id %s, %v", playerEntity.ID, err)
			continue
		}

		player, ok := playerComponent.(*components.Player)
		if !ok {
			log.Error().Msgf("unable to cast component to Player %v", playerComponent)
			continue
		}

		if !util.ContainsClient(excludeClients, player.Client) {
			player.Broadcast(m)
		}
	}
}

// -----------------------------------------------------------------------------

func NewGame() *Game {
	combatSytem := &systems.CombatSystem{}
	movementSystem := &systems.MovementSystem{}

	world := ecs.NewWorld()
	world.AddSystem(combatSytem)
	world.AddSystem(movementSystem)

	defaultRoom, _ := world.GetComponent("1", "Room")

	game := Game{
		defaultRoom:      defaultRoom.(*components.Room),
		players:          make(map[string]*ecs.Entity),
		world:            world,
		AddPlayerChan:    make(chan common.Client),
		RemovePlayerChan: make(chan common.Client),
		CommandChan:      make(chan ClientCommand),
	}

	go game.loop()

	return &game
}
