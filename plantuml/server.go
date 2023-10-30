package plantuml

import (
	"fmt"
	"math/rand"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
	"github.com/godevsig/grepo/fileserver"
)

// Server represents data server
type Server struct {
	ds *as.Server
	lg *log.Logger
	fs *fileserver.FileServer
	rs *RestAPIServer
}

var (
	hostAddr string
	workdir  string
)

// NewServer creates a new server instance.
func NewServer(lg *log.Logger, hostPort, restPort, path string) *Server {
	ip := "0.0.0.0"
	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(0)
	conn := <-c.Discover("builtin", "IPObserver")
	if conn != nil {
		var observedIP string
		err := conn.SendRecv(as.GetObservedIP{}, &observedIP)
		if err == nil {
			ip = observedIP
		}
		conn.Close()
	}

	fs := fileserver.NewFileServer(lg, hostPort, path+"/data", "plantuml database")
	if fs == nil {
		lg.Errorln("create file server failed")
		return nil
	}

	hostAddr = fmt.Sprintf("%s:%s", ip, fs.Port)
	workdir = path

	var opts = []as.Option{as.WithLogger(lg)}
	ds := as.NewServer(opts...).SetPublisher("platform")

	server := &Server{
		rs: NewRestAPIServer(restPort),
		lg: lg,
		ds: ds,
		fs: fs,
	}

	return server
}

// Run runs the server.
func (server *Server) Run() error {
	go server.fs.Start()
	go server.rs.Start()
	defer func() { server.fs.Stop(); server.rs.Stop() }()

	if err := server.ds.Publish("plantuml",
		knownMsgs,
		as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(server.lg) }),
	); err != nil {
		server.lg.Errorf("Create plantuml server failed: %v", err)
		return err
	}

	err := server.ds.Serve()
	if err != nil {
		server.lg.Errorln(err)
	}
	return err
}

// Close shutdown the server.
func (server *Server) Close() {
	server.ds.Close()
}

func randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
