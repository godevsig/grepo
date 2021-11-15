package docit

import (
	_ "embed" //embed: read file

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib/sys/log"
)

// Server represents docit server
type Server struct {
	lg      *log.Logger
	service *as.Server
}

// NewServer creates a new server instance.
func NewServer(lg *log.Logger) *Server {
	var opts = []as.Option{as.WithScope(as.ScopeWAN), as.WithLogger(lg)}
	s := as.NewServer(opts...).SetPublisher("platform")
	return &Server{lg: lg, service: s}
}

// Run runs the server.
func (server *Server) Run() error {
	if err := server.service.Publish("docit",
		knownMsgs,
		as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(server.lg) }),
	); err != nil {
		server.lg.Errorf("create docit server failed: %v", err)
		return err
	}

	err := server.service.Serve()
	if err != nil {
		server.lg.Errorln(err)
	}
	return err
}

// Close shutdown the server.
func (server *Server) Close() {
	server.service.Close()
}
