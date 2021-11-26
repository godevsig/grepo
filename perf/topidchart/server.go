package topidchart

import (
	"fmt"
	"math/rand"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib/sys/log"
	"github.com/godevsig/grepo/util/fileserver"
)

// Server represents data server
type Server struct {
	lg *log.Logger
	ds *as.Server             // data server
	fs *fileserver.FileServer // file server
	cs *chartServer           // chart server
}

var (
	hostAddr string
	dataDir  string
)

// NewServer creates a new server instance.
func NewServer(lg *log.Logger, port, dir string) *Server {
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

	fs := fileserver.NewFileServer(lg, "0", dir, "TOPID DATA")
	if fs == nil {
		lg.Errorln("create file server failed")
		return nil
	}

	cs := newChartServer(lg, ip, port, fs.Port, dir)
	if cs == nil {
		lg.Errorln("create chart server failed")
		return nil
	}

	var opts = []as.Option{as.WithLogger(lg)}
	ds := as.NewServer(opts...).SetPublisher("platform")

	hostAddr = fmt.Sprintf("%s:%s", ip, port)
	dataDir = dir

	server := &Server{
		lg: lg,
		ds: ds,
		fs: fs,
		cs: cs,
	}

	return server
}

// Run runs the server.
func (server *Server) Run() error {
	defer func() { server.cs.stop(); server.fs.Stop() }()

	go server.fs.Start()
	go server.cs.start()

	if err := server.ds.Publish("topidchart",
		knownMsgs,
		as.OnNewStreamFunc(func(ctx as.Context) { ctx.SetContext(server.lg) }),
	); err != nil {
		server.lg.Errorf("create data server failed: %v", err)
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
