package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
	"github.com/godevsig/grepo/recorder"
)

var server *recorder.Server

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	logLevel := flags.String("logLevel", "info", "debug/info/warn/error")
	port := flags.String("port", "0", "set server port, default 0 means alloced by net Listener")
	dir := flags.String("dir", "log", "set directory for saving recorder log data")
	title := flags.String("title", "RECORDER DATA", "set HTML title of file server")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	stream := log.NewStream("")
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("recorder", log.StringToLoglevel(*logLevel))

	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-c.Discover("platform", "recorder")
	if conn != nil {
		conn.Close()
		lg.Warnln("recorder server already running")
		return nil
	}

	fmt.Println("recorder server starting...")
	server = recorder.NewServer(lg, *port, *dir, *title)
	if server == nil {
		return errors.New("create recorder server failed")
	}

	return server.Run()
}

// Stop stops the app
func Stop() {
	fmt.Println("recorder server stopping...")
	server.Close()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
