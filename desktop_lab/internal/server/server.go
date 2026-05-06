package server

import (
	"context"
	"desktop_lab/internal/config"
	"net/http"
)

type Server struct {
	server *http.Server
}

func NewServer(cfg config.ServerConfig, handler http.Handler) *Server {
	return &Server{
		server: &http.Server{
			Addr:           cfg.Host + ":" + cfg.Port,
			Handler:        handler,
			ReadTimeout:    cfg.ReadTimeout,
			WriteTimeout:   cfg.WriteTimeout,
			IdleTimeout:    cfg.IdleTimeout,
			MaxHeaderBytes: cfg.MaxHeaderBytes,
		},
	}
}

func (s *Server) Start() error { return s.server.ListenAndServe() }

func (s *Server) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }
