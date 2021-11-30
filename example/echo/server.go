package echo

import (
	"fmt"
	"sync"
	"sync/atomic"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib/sys/log"
)

// Server is echo server.
type Server struct {
	*as.Server
	lg  *log.Logger
	mgr *statMgr
}

type statMgr struct {
	sync.RWMutex
	lg          *log.Logger
	clients     map[string]struct{}
	subscribers map[chan string]struct{}
	sessionNum  int32
	counter     int64
}

type sessionInfo struct {
	sessionName string
	mgr         *statMgr
}

func (mgr *statMgr) onConnect(netconn as.Netconn) (stop bool) {
	raddr := netconn.RemoteAddr().String()
	mgr.lg.Debugln("on connect from:", raddr)
	mgr.Lock()
	mgr.clients[raddr] = struct{}{}
	mgr.Unlock()

	go func() {
		mgr.RLock()
		for ch := range mgr.subscribers {
			mgr.RUnlock()
			ch <- raddr
			mgr.RLock()
		}
		mgr.RUnlock()
	}()

	return false
}

func (mgr *statMgr) onDisconnect(netconn as.Netconn) {
	raddr := netconn.RemoteAddr().String()
	mgr.lg.Debugln("on disconnect from:", raddr)
	mgr.Lock()
	delete(mgr.clients, raddr)
	mgr.Unlock()
}

func (mgr *statMgr) onNewStream(ctx as.Context) {
	mgr.lg.Debugln("on new stream")
	sessionName := fmt.Sprintf("yours echo.v1.0 from %d", atomic.AddInt32(&mgr.sessionNum, 1))
	ctx.SetContext(&sessionInfo{sessionName, mgr})
}

// NewServer creates a new server instance.
func NewServer(lg *log.Logger) *Server {
	s := as.NewServer(as.WithLogger(lg)).SetPublisher(Publisher)
	mgr := &statMgr{
		lg:          lg,
		clients:     make(map[string]struct{}),
		subscribers: make(map[chan string]struct{}),
	}
	if err := s.Publish(ServiceEcho,
		echoKnownMsgs,
		as.OnNewStreamFunc(mgr.onNewStream),
		as.OnConnectFunc(mgr.onConnect),
		as.OnDisconnectFunc(mgr.onDisconnect),
	); err != nil {
		lg.Errorln(err)
		return nil
	}
	return &Server{s, lg, mgr}
}

// Run runs the server.
func (s *Server) Run() error {
	return s.Serve()
}

// Stop stops the server.
func (s *Server) Stop() {
	s.Close()
	s.lg.Infof("echo server has served %d requests\n", s.mgr.counter)
}
