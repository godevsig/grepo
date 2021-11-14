package markdown

import (
	_ "embed" //embed: read file

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib-sys/log"
)

// Server represents markdown server
type Server struct {
	lg *log.Logger
	ms *as.Server
}

// NewServer creates a new server instance.
func NewServer(lg *log.Logger) *Server {
	var opts = []as.Option{as.WithScope(as.ScopeWAN), as.WithLogger(lg)}
	ms := as.NewServer(opts...).SetPublisher("platform")

	return &Server{lg: lg, ms: ms}
}

// Run runs the server.
func (server *Server) Run() error {

	if err := server.ms.Publish("markdown",
		knownMsgs,
		as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(server.lg) }),
	); err != nil {
		server.lg.Errorf("create markdown server failed: %v", err)
		return err
	}

	err := server.ms.Serve()
	if err != nil {
		server.lg.Errorln(err)
	}
	return err
}

// Close shutdown the server.
func (server *Server) Close() {
	server.ms.Close()
}
