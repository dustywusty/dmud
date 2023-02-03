package net

func NewServer(config *ServerConfig) *Server {
  return &Server{
    host: config.Host,
    port: config.Port,
  }
}