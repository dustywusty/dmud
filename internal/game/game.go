package game

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

///////////////////////////////////////////////////////////////////////////////////////////////
// Game
//

type Game struct {
	defaultRoom      *components.RoomComponent
	players          map[string]*components.PlayerComponent
	playersMu        sync.Mutex
	world            *ecs.World
	AddPlayerChan    chan common.Client
	RemovePlayerChan chan common.Client
	CommandChan      chan ClientCommand
}

func NewGame() *Game {
	world := ecs.NewWorld()
	defaultRoom, _ := world.GetComponent("1", "RoomComponent")

	game := Game{
		defaultRoom:      defaultRoom.(*components.RoomComponent),
		players:          make(map[string]*components.PlayerComponent),
		world:            world,
		AddPlayerChan:    make(chan common.Client),
		RemovePlayerChan: make(chan common.Client),
		CommandChan:      make(chan ClientCommand),
	}

	go game.loop()

	return &game
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func (g *Game) HandleConnect(c common.Client) {
	go func() {
		log.Printf("New connection from %s", c.RemoteAddr())

		accounts, err := loadAccountsFromFile("./resources/accounts.json")
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		exists := false
		name, _ := g.queryPlayerName(c)

		var a Account
		for _, a = range accounts {
			if a.Name == name {
				exists = true
			}
		}

		password, _ := g.queryPlayerPassword(c, exists)

		if exists {
			log.Printf("Account %s password %s", name, password)

			err := bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(password))
			if err != nil {
				log.Printf("Error comparing password: %v", err)
				c.CloseConnection()
				return
			}
		} else {
			account := Account{
				Name:     name,
				Password: util.HashAndSalt(password),
			}
			accounts = append(accounts, account)
			saveAccountsToFile("./resources/accounts.json", accounts)
		}

		playerComponent := &components.PlayerComponent{
			Client: c,
			Name:   name,
			Room:   g.defaultRoom,
		}
		g.addPlayer(playerComponent)
	}()
}

func (g *Game) HandleDisconnect(c common.Client) {
	g.RemovePlayerChan <- c
}

func (g *Game) RemovePlayer(c common.Client) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	player, err := g.getPlayer(c)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	g.world.RemoveEntity(player.EntityID)
	delete(g.players, player.Name)

	g.messageAllPlayers(fmt.Sprintf("%s has left the game.", player.Name), c)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Private
//

func (g *Game) addPlayer(p *components.PlayerComponent) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	g.players[p.Name] = p

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)
	g.world.AddComponent(playerEntity, p)

	g.messageAllPlayers(fmt.Sprintf("%s has joined the game.", p.Name), p.Client)

	p.Client.SendMessage(util.WelcomeBanner)
	p.Client.SendMessage(g.defaultRoom.Description)

	go p.Client.HandleRequest()

	log.Info().Msg(fmt.Sprintf("Player %s added", p.Name))
}

func containsClient(clients []common.Client, client common.Client) bool {
	for _, c := range clients {
		if c == client {
			return true
		}
	}
	return false
}

func (g *Game) getPlayer(c common.Client) (*components.PlayerComponent, error) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	for _, player := range g.players {
		if player.Client == c {
			return player, nil
		}
	}
	return nil, fmt.Errorf("Player not found")
}

func (g *Game) handleCommand(c ClientCommand) {
	command := c.Command
	client := c.Client
	player, err := g.getPlayer(client)

	if err != nil {
		fmt.Println("Error getting player component:", err)
		return
	}

	switch c.Command.Cmd {
	case "exit":
		g.handleExit(player, command)
	case "shout":
		g.handleShout(player, command)
	default:
		g.handleUnknownCommand(player, command)
	}
}

func (g *Game) handleExit(player *components.PlayerComponent, command Command) {
	player.Client.CloseConnection()

	g.messageAllPlayers(fmt.Sprintf("%s has left the game.", player.Name), player.Client)
}

func (g *Game) handleShout(player *components.PlayerComponent, command Command) {
	message := fmt.Sprintf("%s shouts %s", player.Name, strings.Join(command.Args, " "))
	g.messageAllPlayers(message, player.Client)
	player.Client.SendMessage(fmt.Sprintf("You shout %s", strings.Join(command.Args, " ")))
}

func (g *Game) handleUnknownCommand(player *components.PlayerComponent, command Command) {
	player.Client.SendMessage(fmt.Sprintf("What do you mean, \"%s\"?", command.Cmd))
}

func (g *Game) loop() {
	updateTicker := time.NewTicker(10 * time.Millisecond)
	defer updateTicker.Stop()

	for {
		select {
		case client := <-g.AddPlayerChan:
			g.HandleConnect(client)
		case client := <-g.RemovePlayerChan:
			g.RemovePlayer(client)
		case command := <-g.CommandChan:
			g.handleCommand(command)
		case <-updateTicker.C:
			g.world.Update()
		}
	}
}

func (g *Game) messageAllPlayers(m string, excludeClients ...common.Client) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	for _, player := range g.players {
		if !containsClient(excludeClients, player.Client) {
			player.Client.SendMessage(m)
		}
	}
}

func (g *Game) queryPlayerName(client common.Client) (string, error) {
	var name string
	var err error

	for {
		log.Printf("Querying player name")
		client.SendMessage("What do we call you?")

		name, err = client.GetMessage(32)
		if err != nil {
			log.Error().Err(err).Msg("")
			return "", err
		}

		if len(name) > 32 || len(name) <= 0 || !util.IsAlphaNumeric(name) {
			continue
		}
		break
	}
	return name, nil
}

func (g *Game) queryPlayerPassword(c common.Client, exists bool) (string, error) {
	var password string
	var err error

	for {
		c.SendMessage("What is your password?")

		password, err = c.GetMessage(256)
		if err != nil {
			log.Error().Err(err).Msg("")
			return "", err
		}

		if len(password) <= 0 || len(password) > 256 {
			continue
		}

		if !exists {
			c.SendMessage("Please confirm your password.")
			confirmPassword, err := c.GetMessage(256)
			if err != nil {
				log.Error().Err(err).Msg("")
				return "", err
			}

			if confirmPassword != password {
				c.SendMessage("Passwords do not match. Please try again.")
				continue
			}
			break
		}
		break
	}
	return password, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

type Account struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func loadAccountsFromFile(filename string) ([]Account, error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var accounts []Account

	err = json.Unmarshal(byteValue, &accounts)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func saveAccountsToFile(filename string, accounts []Account) error {
	data, err := json.Marshal(accounts)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
