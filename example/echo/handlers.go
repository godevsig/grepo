package echo

import (
	"sync/atomic"
	"time"

	as "github.com/godevsig/adaptiveservice"
)

// Handle handles msg.
func (msg Request) Handle(stream as.ContextStream) (reply interface{}) {
	si := stream.GetContext().(*sessionInfo)
	msg.Msg += "!"
	msg.Num++
	atomic.AddInt64(&si.mgr.counter, 1)
	time.Sleep(time.Second / 2)
	return Reply{msg, si.sessionName}
}

// Handle handles msg.
func (msg SubWhoElseEvent) Handle(stream as.ContextStream) (reply interface{}) {
	si := stream.GetContext().(*sessionInfo)
	ch := make(chan string, 1)
	si.mgr.Lock()
	si.mgr.subscribers[ch] = struct{}{}
	si.mgr.Unlock()
	go func() {
		for {
			addr := <-ch
			if err := stream.Send(addr); err != nil {
				si.mgr.Lock()
				delete(si.mgr.subscribers, ch)
				si.mgr.Unlock()
				return
			}
		}
	}()
	return 0
}

// Handle handles msg.
func (msg WhoElse) Handle(stream as.ContextStream) (reply interface{}) {
	si := stream.GetContext().(*sessionInfo)
	var addrs string
	si.mgr.RLock()
	for client := range si.mgr.clients {
		addrs += " " + client
	}
	si.mgr.RUnlock()
	return addrs
}

var echoKnownMsgs = []as.KnownMessage{
	Request{},
	SubWhoElseEvent{},
	WhoElse{},
}
