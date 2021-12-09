package asbench

import (
	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib/sys/log"
)

// Server is echo server.
type Server struct {
	*as.Server
}

// NewServer creates a new server instance.
func NewServer(lg *log.Logger) *Server {
	s := as.NewServer(as.WithLogger(lg)).SetPublisher("benchmark").DisableMsgTypeCheck()
	if err := s.Publish("asbench",
		knownMsgs,
	); err != nil {
		lg.Errorln(err)
		return nil
	}
	return &Server{s}
}

// Run runs the server.
func (s *Server) Run() error {
	return s.Serve()
}

// Stop stops the server.
func (s *Server) Stop() {
	s.Close()
}
