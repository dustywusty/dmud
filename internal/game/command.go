package game

import "dmud/internal/common"

type Command struct {
	Cmd    string
	Args   []string
	Client common.Client
}
