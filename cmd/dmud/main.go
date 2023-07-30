package main

import (
	"dmud/internal/net"
	"dmud/internal/util"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
		TCPHost: "127.0.0.1",
		TCPPort: "3333",
	})
	go server.Run()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt

	log.Info().Msg("Received interrupt signal, shutting down...")
	server.Shutdown()
	log.Info().Msg("Server successfully shut down. Exiting.")
}
