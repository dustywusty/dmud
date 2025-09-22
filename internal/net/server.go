package net

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"dmud/internal/common"
	"dmud/internal/game"

	"github.com/gorilla/websocket"
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

	wsServer *http.Server
	wsHost   string
	wsPort   string

	wsMux     *http.ServeMux
	wsMuxOnce sync.Once
}

func (s *Server) Run() {
	var wg sync.WaitGroup
	s.game = game.NewGame()

	started := 0

	if s.tcpHost != "" && s.tcpPort != "" {
		wg.Add(1)
		started++
		go func() {
			s.runTCPListener()
			wg.Done()
		}()
	}

	if s.wsHost != "" && s.wsPort != "" {
		wg.Add(1)
		started++
		go func() {
			s.runWebSocketServer()
			wg.Done()
		}()
	}

	if started == 0 {
		log.Fatal().Msg("No listeners configured")
	}

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

	if s.wsServer != nil {
		if err := s.wsServer.Shutdown(context.Background()); err != nil {
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
				conn: conn,
				game: s.game,
			}

			s.connectionMu.Lock()
			s.connections[remoteAddr] = client
			s.connectionMu.Unlock()

			s.game.AddPlayerChan <- client
		}
	}()

	<-done
}

func (s *Server) runWebSocketServer() {
	done := make(chan bool)

	s.wsMuxOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			if !websocket.IsWebSocketUpgrade(r) {
				http.Error(w, "websocket upgrade required", http.StatusUpgradeRequired)
				return
			}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Error().Err(err).Msg("websocket upgrade failed")
				return
			}
			remoteAddr := conn.RemoteAddr().String()
			log.Info().Msgf("Accepted WebSocket connection from %s", remoteAddr)

			client := &WSClient{conn: conn, status: common.Connected, game: s.game}

			s.connectionMu.Lock()
			s.connections[remoteAddr] = client
			s.connectionMu.Unlock()

			// On HTTP handler exit (which happens when the TCP conn dies), drop it
			defer func() {
				s.connectionMu.Lock()
				delete(s.connections, remoteAddr)
				s.connectionMu.Unlock()
			}()

			s.game.AddPlayerChan <- client
		})
		mux.Handle("/", s.webFileServer())
		s.wsMux = mux
	})

	s.wsServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", s.wsHost, s.wsPort),
		Handler: s.wsMux, // reuse same mux every time
	}

	log.Info().Msgf("Listening WebSocket on %s:%s", s.wsHost, s.wsPort)

	go func() {
		err := s.wsServer.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start WebSocket server")
		} else {
			log.Info().Msg("WebSocket server stopped")
		}
		done <- true
	}()

	<-done
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
