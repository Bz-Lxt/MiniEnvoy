package admin

import (
	"context"
	"net"
	"net/http"
	"time"
)

type Server struct {
	http *http.Server
}

func Listen(bind string, h http.Handler) (*Server, error) {
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}
	s := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() { _ = s.Serve(ln) }()
	return &Server{http: s}, nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}
