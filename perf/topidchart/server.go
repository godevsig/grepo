package topidchart

import (
	"fmt"
	"math/rand"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib/sys/log"
)

// Server represents data server
type Server struct {
	ip   string
	port string
	dir  string
	lg   *log.Logger
	ds   *as.Server   // data server
	fs   *fileServer  // file server
	cs   *chartServer // chart server
}

var (
	hostAddr string
	dataDir  string
)

// NewServer creates a new server instance.
func NewServer(lg *log.Logger, port, dir string) *Server {
	c := as.NewClient().SetDiscoverTimeout(3)
	conn := <-c.Discover("builtin", "IPObserver")
	if conn == nil {
		lg.Errorln("IPObserver service not found")
		return nil
	}
	var ip string
	if err := conn.SendRecv(as.GetObservedIP{}, &ip); err != nil {
		lg.Errorln("get observed ip failed: %v", err)
		return nil
	}
	conn.Close()

	fs := newFileServer(lg, dir)
	if fs == nil {
		lg.Errorln("create file server failed")
		return nil
	}

	cs := newChartServer(lg, ip, port, fs.port, dir)
	if cs == nil {
		lg.Errorln("create chart server failed")
		return nil
	}

	var opts = []as.Option{as.WithLogger(lg)}
	ds := as.NewServer(opts...).SetPublisher("platform")

	hostAddr = fmt.Sprintf("%s:%s", ip, port)
	dataDir = dir

	server := &Server{
		ip:   ip,
		port: port,
		dir:  dir,
		lg:   lg,
		ds:   ds,
		fs:   fs,
		cs:   cs,
	}

	return server
}

// Run runs the server.
func (server *Server) Run() error {
	defer func() { server.cs.stop(); server.fs.stop() }()

	go server.fs.start()
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
