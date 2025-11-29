package main

import (
	"os"
	"os/signal"
	"syscall"

	"dmud/internal/components"
	"dmud/internal/net"
	"dmud/internal/util"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	// Useful options when debugging
	//
	// Caller().
	// Int("pid", os.Getpid()).
	// Str("go_version", runtime.Version()).

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05.000"}).
		Level(zerolog.TraceLevel).
		With().
		Timestamp().
		Str("thread_id", util.GetGID()).
		Logger()
	log.Logger = logger

	// Load NPC templates from JSON
	if err := components.LoadNPCTemplates("./resources/npcs.json"); err != nil {
		log.Fatal().Err(err).Msg("Failed to load NPC templates")
	}

	// Initialize quest dialogues
	components.InitializeQuests()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"

	}
	server := net.NewServer(&net.ServerConfig{
		WSHost: "0.0.0.0", WSPort: port,
	})

	go server.Run()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt

	log.Info().Msg("Received interrupt signal, shutting down...")

	server.Shutdown()
}
