package net

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"dmud/internal/common"
	"dmud/internal/game"

	"github.com/rs/zerolog/log"
)

type ServerConfig struct {
	TCPHost string
	TCPPort string

	WSHost string
	WSPort string
}

type Server struct {
	connectionMu sync.Mutex
	connections  map[string]common.Client

	game *game.Game

	tcpListener net.Listener
	tcpHost     string
	tcpPort     string

	httpServer *http.Server
	wsHost     string
	wsPort     string
}

func NewServer(config *ServerConfig) *Server {
	return &Server{
		tcpHost:     config.TCPHost,
		tcpPort:     config.TCPPort,
		wsHost:      config.WSHost,
		wsPort:      config.WSPort,
		connections: make(map[string]common.Client),
	}
}

func (s *Server) Run() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		s.runTCPListener()
		wg.Done()
	}()

	go func() {
		s.runWebSocketServer()
		wg.Done()
	}()

	s.game = game.NewGame()

	wg.Wait()
}

func (s *Server) Shutdown() {
	s.connectionMu.Lock()
	for _, client := range s.connections {
		client.CloseConnection()
	}
	s.connectionMu.Unlock()

	if s.tcpListener != nil {
		if err := s.tcpListener.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close TCP listener")
		} else {
			log.Info().Msg("TCP listener successfully closed")
		}
	}

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown HTTP server")
		} else {
			log.Info().Msg("WebSocket server successfully shut down")
		}
	}
}

func (s *Server) runTCPListener() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", s.tcpHost, s.tcpPort))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	s.tcpListener = listener

	log.Info().Msgf("Listening TCP on %s:%s", s.tcpHost, s.tcpPort)

	done := make(chan bool)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
					log.Info().Msg("TCP listener stopped")
				} else {
					log.Error().Err(err).Msg("")
				}
				done <- true
				return
			}

			remoteAddr := conn.RemoteAddr().String()
			log.Info().Msgf("Accepted TCP connection from %s", remoteAddr)

			client := &TCPClient{
				conn:   conn,
				game:   s.game,
				reader: bufio.NewReader(conn),
			}

			s.connectionMu.Lock()
			s.connections[remoteAddr] = client
			s.connectionMu.Unlock()

			s.game.AddPlayerChan <- client

			go client.HandleRequest()
		}
	}()

	// This will block until done is signaled after listener.Accept() fails due to listener being closed
	<-done
}

func (s *Server) runWebSocketServer() {
	done := make(chan bool)

	s.httpServer = &http.Server{
		Addr: fmt.Sprintf("%s:%s", s.wsHost, s.wsPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to set websocket upgrade")
				return
			}

			remoteAddr := conn.RemoteAddr().String()
			log.Info().Msgf("Accepted WebSocket connection from %s", remoteAddr)

			client := &WSClient{
				conn: conn,
				game: s.game,
			}

			s.connectionMu.Lock()
			s.connections[remoteAddr] = client
			s.connectionMu.Unlock()

			s.game.AddPlayerChan <- client

			go client.HandleRequest()
		}),
	}

	log.Info().Msgf("Listening WebSocket on %s:%s", s.wsHost, s.wsPort)

	go func() {
		err := s.httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start WebSocket server")
		} else {
			log.Info().Msg("WebSocket server stopped")
		}
		done <- true
	}()

	// This will block until done is signaled after httpServer.ListenAndServe() fails due to httpServer being shut down
	<-done
}
