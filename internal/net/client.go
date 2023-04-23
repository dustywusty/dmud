package net

import (
	"fmt"
	"math/rand"
	"time"
  "bufio"
  "dmud/internal/game"
  "log"
  "net"
  "strings"
)

type Client struct {
  conn net.Conn
  game *game.Game
  name string
}

var nouns = []string{
  "dragon",
  "wizard",
  "elf",
  "knight",
  "orc",
  "vampire",
  "troll",
  "dwarf",
  "demon",
  "phoenix",
  "hydra",
  "behemoth",
  "chimera",
  "gargoyle",
  "harpy",
  "minotaur",
  "naga",
  "specter",
  "yeti",
  "zombie",
  "banshee",
  "kraken",
  "mermaid",
  "unicorn",
  "wyvern",
  "cyclops",
  "genie",
  "golem",
  "leviathan",
  "ogre",
  "sphinx",
  "titan",
}

var verbs1 = []string{
  "slaying",
  "vanquishing",
  "conquering",
  "fighting",
  "battling",
  "defeating",
  "crushing",
  "smashing",
  "devastating",
  "overcoming",
  "destroying",
  "eliminating",
  "demolishing",
  "annihilating",
  "decimating",
  "obliterating",
  "eradicating",
  "exterminating",
  "erasing",
  "wiping out",
  "purging",
  "cleansing",
  "scorched-earth",
  "ruination",
  "carnage",
  "bloodshed",
  "massacre",
  "slaughter",
  "butchery",
  "genocide",
  "apocalypse",
}

var verbs2 = []string{
  "conjuring",
  "summoning",
  "evoking",
  "casting",
  "enchanting",
  "bewitching",
  "hexing",
  "cursing",
  "jinxing",
  "incanting",
  "chanting",
  "ensnaring",
  "entrapping",
  "imprisoning",
  "controlling",
  "dominating",
  "subjugating",
  "enslaving",
  "binding",
  "constricting",
  "imposing",
  "commanding",
  "influencing",
  "persuading",
  "beguiling",
  "mesmerizing",
  "hypnotizing",
  "dazzling",
  "blinding",
  "fascinating",
}

var verbs3 = []string{
  "exploring",
  "adventuring",
  "questing",
  "searching",
  "hunting",
  "tracking",
  "discovering",
  "uncovering",
  "revealing",
  "finding",
  "retrieving",
  "obtaining",
  "acquiring",
  "collecting",
  "hoarding",
  "accumulating",
  "amassing",
  "stockpiling",
  "gathering",
  "scavenging",
  "salvaging",
  "picking up",
  "seeking",
  "pursuing",
  "chasing",
  "following",
  "hounding",
  "stalking",
  "trailing",
  "ambushing",
  "surprising",
}

func (client *Client) Name() string {
  return client.name
}

func (client *Client) RemoteAddr() string {
  return client.conn.RemoteAddr().String()
}

func (client *Client) SendMessage(msg string) {
	client.conn.Write([]byte(msg))
}

func (client *Client) generateRandomName() string {
  rand.Seed(time.Now().UnixNano())
  noun := nouns[rand.Intn(len(nouns))]
  verb1 := verbs1[rand.Intn(len(verbs1))]
  verb2 := verbs2[rand.Intn(len(verbs2))]
  return verb1 + "-" + verb2 + "-" + noun
}


func (client *Client) handleRequest() {
  client.SendMessage(fmt.Sprintf("Welcome to the server, %s!\n\n> ", client.name))

  reader := bufio.NewReader(client.conn)

  for {
    message, err := reader.ReadString('\n')

    if err != nil {
      client.conn.Close()
      client.game.RemovePlayer(client)
      return
    }

    cmd := parseCommand(message)
    if cmd != nil {
      log.Printf("Received command: %s, args: %s", cmd.Name, cmd.Arguments)
			client.SendMessage("\n")
      client.game.ExecuteCommand(client, cmd)
    } else {
      log.Printf("Invalid command: %s", message)
    }

    client.SendMessage("\n\n> ")
  }
}

func parseCommand(message string) *game.Command {
  words := strings.Fields(message)
  if len(words) == 0 {
    return nil
  }

  cmd := &game.Command{
    Name:      strings.ToLower(words[0]),
    Arguments: words[1:],
  }

  return cmd
}
