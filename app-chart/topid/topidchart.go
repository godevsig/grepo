package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/godevsig/grepo/lib-sys/log"
	"github.com/godevsig/grepo/srv-chart/topid"
)

var server *topid.DataServer

// Start starts the service
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	logLevel := flags.String("logLevel", "info", "set log level")
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
		topid.Parse(*parsefile)
		return nil
	}

	stream := log.NewStream("")
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("topidchart", log.StringToLoglevel(*logLevel))

	fmt.Println("topid chart server starting...")
	server := topid.NewServer(lg, *port, *dir)
	if server == nil {
		return errors.New("create topid chart server failed")
	}

	server.Start()

	return nil
}

// Stop stops the service
func Stop() {
	fmt.Println("topid chart server stopping...")
	server.Stop()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
