package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
	topid "github.com/godevsig/grepo/topidchart"
)

var server *topid.Server

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	logLevel := flags.String("logLevel", "info", "debug/info/warn/error")
	dir := flags.String("dir", "topidata", "set directory for saving topid raw data")
	port := flags.String("port", "9998", "set port for visiting chart http server")
	parsefile := flags.String("parse", "", "parse file")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	if len(*parsefile) != 0 {
		return topid.ParseFile(*parsefile)
	}

	stream := log.NewStream("")
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("topidchart", log.StringToLoglevel(*logLevel))

	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-c.Discover("platform", "topidchart")
	if conn != nil {
		conn.Close()
		lg.Warnln("topid chart server already running")
		return nil
	}

	fmt.Println("topid chart server starting...")
	server = topid.NewServer(lg, *port, *dir)
	if server == nil {
		return errors.New("create topid chart server failed")
	}

	return server.Run()
}

// Stop stops the app
func Stop() {
	fmt.Println("topid chart server stopping...")
	server.Close()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
