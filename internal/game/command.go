package game

import "dmud/internal/common"

type ClientCommand struct {
	Client  common.Client
	Command Command
}

type Command struct {
	Cmd  string
	Args []string
}
