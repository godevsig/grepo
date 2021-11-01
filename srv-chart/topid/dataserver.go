package topid

import (
	"fmt"
	"math/rand"
	"time"

	as "github.com/godevsig/adaptiveservice"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var server *as.Server

func randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// Run runs the server.
func Run(opts []as.Option) {
	client := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-client.Discover("platform", "topidchart")
	if conn != nil {
		conn.Close()
		panic("topid chart server already running, exit!")
	}

	server = as.NewServer(opts...).SetPublisher("platform")
	if err := server.Publish("topidchart",
		knownMsgs,
	); err != nil {
		fmt.Println(err)
		return
	}

	go startFileServer()
	go startChartServer()

	if err := server.Serve(); err != nil {
		fmt.Println(err)
	}
}

// Shutdown shutdown the server.
func Shutdown() {
	stopChartServer()
	stopFileServer()
	server.Close()
}
