package server

import (
	"fmt"

	"go.uber.org/fx"
)

type Server struct {
	// contains filtered or unexported fields
}

type serverParams struct {
	fx.In
	// contains filtered or unexported fields
}

// NewServer creates a new server instance.
func NewServer() *Server {
	return &Server{}
}

func (s *Server) ListenAndServe() error {
	fmt.Println("Server is listening on port 8080")
	return nil
}
