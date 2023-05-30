package main

import (
	"dmud/internal/net"
	"dmud/internal/util"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s (%s)", i, util.GetGID())
	}

	log.Logger = log.Output(output)

	server := net.NewServer(&net.ServerConfig{
		Host: "localhost",
		Port: "3333",
	})
	server.Run()
}
