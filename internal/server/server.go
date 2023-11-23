package server

type Server struct {
	// contains filtered or unexported fields
}

// NewServer creates a new server instance.
func NewServer() *Server {
	return &Server{}
}

func (s *Server) ListenAndServe() error {
	return nil
}
