package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	_ "embed" //embed: read file

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/lib-sys/log"
	"github.com/godevsig/grepo/srv-chart/markdown"
)

var server *markdown.Server

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	logLevel := flags.String("logLevel", "info", "debug/info/warn/error")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	stream := log.NewStream("")
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("markdown", log.StringToLoglevel(*logLevel))

	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-c.Discover("platform", "markdown")
	if conn != nil {
		conn.Close()
		lg.Warnln("markdown server already running")
		return nil
	}

	fmt.Println("markdown server starting...")
	server = markdown.NewServer(lg)
	if server == nil {
		return errors.New("create markdown server failed")
	}

	return server.Run()
}

// Stop stops the app
func Stop() {
	fmt.Println("markdown server stopping...")
	server.Close()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
