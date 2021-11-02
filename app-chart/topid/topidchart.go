package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/godevsig/grepo/lib-sys/log"
	"github.com/godevsig/grepo/srv-chart/topid"
)

var (
	dir       string
	port      string
	parsefile string
	logLevel  string
)

var server *topid.DataServer

// Start starts the service
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.StringVar(&logLevel, "logLevel", "info", "set log level")
	flags.StringVar(&dir, "dir", "topidata", "set directory for saving topid raw data")
	flags.StringVar(&port, "port", "9998", "set port for visiting chart http server")
	flags.StringVar(&parsefile, "parse", "", "parse file")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	if len(parsefile) != 0 {
		topid.Parse(parsefile)
		return nil
	}

	stream := log.NewStream("")
	defer stream.Close()
	stream.SetOutputter(os.Stdout)
	level := log.Linfo
	switch logLevel {
	case "debug":
		level = log.Ldebug
	case "info":
		level = log.Linfo
	case "warn":
		level = log.Lwarn
	case "error":
		level = log.Lerror
	}
	lg := stream.NewLogger("topidchart", level)
	defer lg.Close()

	server := topid.NewServer(lg, port, dir)
	if server == nil {
		return errors.New("create topid chart server failed")
	}
	fmt.Println("topid chart server starting...")
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
