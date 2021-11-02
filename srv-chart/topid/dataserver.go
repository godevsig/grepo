package topid

import (
	"fmt"
	"math/rand"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib-sys/log"
)

// DataServer represents data server
type DataServer struct {
	ip     string
	port   string
	dir    string
	server *as.Server
	lg     *log.Logger
}

var (
	hostAddr string
	dataDir  string
	fs       *fileServer
	cs       *chartServer
)

// NewServer creates a new server instance.
func NewServer(lg *log.Logger, port, dir string) *DataServer {
	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-c.Discover("platform", "topidchart")
	if conn != nil {
		conn.Close()
		lg.Errorln("topid chart server already running!")
		return nil
	}

	var opts = []as.Option{as.WithScope(as.ScopeWAN), as.WithLogger(lg)}
	s := as.NewServer(opts...).SetPublisher("platform")
	if err := s.Publish("topidchart",
		knownMsgs,
	); err != nil {
		lg.Errorln(err)
		return nil
	}

	c = as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn = <-c.Discover("builtin", "IPObserver")
	if conn == nil {
		lg.Errorln("IPObserver service not found!")
		return nil
	}
	var ip string
	if err := conn.SendRecv(as.GetObservedIP{}, &ip); err != nil {
		lg.Errorln("get observed ip failed!")
		return nil
	}
	conn.Close()

	hostAddr = fmt.Sprintf("%s:%s", ip, port)
	dataDir = dir

	ds := &DataServer{
		ip:     ip,
		port:   port,
		dir:    dir,
		server: s,
		lg:     lg,
	}

	return ds
}

// Start runs the server.
func (ds *DataServer) Start() {
	fs = newFileServer(ds.lg, ds.dir)
	go fs.start()

	cs = newChartServer(ds.lg, ds.ip, ds.port, fs.port, ds.dir)
	go cs.start()

	if err := ds.server.Serve(); err != nil {
		ds.lg.Errorln(err)
	}
}

// Stop shutdown the server.
func (ds *DataServer) Stop() {
	cs.stop()
	fs.stop()
	ds.server.Close()
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
