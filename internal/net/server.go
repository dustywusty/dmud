package net

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	"dmud/internal/common"
	"dmud/internal/game"

	"github.com/rs/zerolog/log"
)

type ServerConfig struct {
	Host string
	Port string
}

type Server struct {
	connectionMu sync.Mutex
	connections  map[string]common.Client
	game         *game.Game
	host         string
	port         string
}

func NewServer(config *ServerConfig) *Server {
	return &Server{
		host:        config.Host,
		port:        config.Port,
		connections: make(map[string]common.Client),
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func (s *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", s.host, s.port))
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Error().Err(err).Msg("")
		}
	}(listener)

	log.Info().Msgf("Listening on %s:%s", s.host, s.port)

	s.game = game.NewGame()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		remoteAddr := conn.RemoteAddr().String()
		log.Info().Msgf("Accepted connection from %s", remoteAddr)

		client := &TCPClient{
			conn:   conn,
			game:   s.game,
			reader: bufio.NewReader(conn),
		}

		s.connectionMu.Lock()
		s.connections[remoteAddr] = client
		s.connectionMu.Unlock()

		s.game.AddPlayerChan <- client
	}
}
