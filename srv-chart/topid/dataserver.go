package topid

import (
	"fmt"
	"math/rand"
	"time"

	as "github.com/godevsig/adaptiveservice"
)

// DataServer represents data server
type DataServer struct {
	ip     string
	port   string
	dir    string
	server *as.Server
}

var (
	hostAddr string
	dataDir  string
	fs       *fileServer
	cs       *chartServer
)

// NewServer creates a new server instance.
func NewServer(opts []as.Option, port, dir string) *DataServer {
	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-c.Discover("platform", "topidchart")
	if conn != nil {
		conn.Close()
		fmt.Println("topid chart server already running!")
		return nil
	}

	s := as.NewServer(opts...).SetPublisher("platform")
	if err := s.Publish("topidchart",
		knownMsgs,
	); err != nil {
		fmt.Println(err)
		return nil
	}

	c = as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn = <-c.Discover("builtin", "IPObserver")
	if conn == nil {
		fmt.Println("IPObserver service not found!")
		return nil
	}
	var ip string
	if err := conn.SendRecv(as.GetObservedIP{}, &ip); err != nil {
		fmt.Println("get observed ip failed!")
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
	}

	return ds
}

// Start runs the server.
func (ds *DataServer) Start() {
	fs = newFileServer(ds.dir)
	go fs.start()

	cs = newChartServer(ds.ip, ds.port, fs.port, ds.dir)
	go cs.start()

	if err := ds.server.Serve(); err != nil {
		fmt.Println(err)
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
