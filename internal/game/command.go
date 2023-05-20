package game

import (
	"dmud/internal/ecs"
)

type Command struct {
	Name      string
	Arguments []string
	PlayerID  string
}

func (c *Command) Execute(world *ecs.World) {
	switch c.Name {
	case "say":
		c.executeSayCommand(world)
	default:
		// Handle other commands or log an invalid command message
	}
}

func (c *Command) executeSayCommand(world *ecs.World) {

}
