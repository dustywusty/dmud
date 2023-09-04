package main

import (
	"dmud/internal/net"
	"dmud/internal/util"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05.000"}).
		Level(zerolog.TraceLevel).
		With().
		Timestamp().
		// Caller().
		// Int("pid", os.Getpid()).
		// Str("go_version", runtime.Version()).
		Str("thread_id", util.GetGID()).
		Logger()

	log.Logger = logger

	server := net.NewServer(&net.ServerConfig{
		WSHost: "localhost",
		WSPort: "8080",
	})

	go server.Run()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt

	log.Info().Msg("Received interrupt signal, shutting down...")

	server.Shutdown()
}
