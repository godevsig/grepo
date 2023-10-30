package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
	"github.com/godevsig/grepo/plantuml"
)

var server *plantuml.Server

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	logLevel := flags.String("logLevel", "info", "debug/info/warn/error")
	hostPort := flags.String("hostport", "0", "set server port, default 0 means alloced by net Listener")
	restPort := flags.String("restport", "8364", "set restful api port, default 8364")

	path := flags.String("workdir", "/tmp/plantuml", "directory path of workdir, used to save plantuml jar, text and svg. (default \"/tmp/plantuml\")")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	stream := log.NewStream("")
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("plantuml", log.StringToLoglevel(*logLevel))

	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-c.Discover("platform", "plantuml")
	if conn != nil {
		conn.Close()
		lg.Warnln("plantuml server already running")
		return nil
	}

	fmt.Println("plantuml server starting...")
	server = plantuml.NewServer(lg, *hostPort, *restPort, *path)
	if server == nil {
		return errors.New("create plantuml server failed")
	}

	return server.Run()
}

// Stop stops the app
func Stop() {
	fmt.Println("plantuml server stopping...")
	server.Close()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
