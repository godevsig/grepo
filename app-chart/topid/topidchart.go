package main

import (
	"flag"
	"fmt"
	"os"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/srv-chart/topid"
	//"github.com/godevsig/grepo/lib-sys/log"
)

var (
	dir       string
	port      string
	parsefile string
	debug     bool
)

var server *topid.DataServer

// Start starts the service
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	flags.BoolVar(&debug, "debug", false, "enable debug")
	flags.StringVar(&dir, "dir", "topidata", "directory for saving topid raw data")
	flags.StringVar(&port, "port", "9998", "port for visiting chart http server")
	flags.StringVar(&parsefile, "parse", "", "parse file")

	if err = flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return
	}

	if len(parsefile) != 0 {
		topid.Parse(parsefile)
		return
	}

	var opts = []as.Option{as.WithScope(as.ScopeWAN)}
	if debug {
		opts = append(opts, as.WithLogger(as.LoggerAll{}))
	}

	server := topid.NewServer(opts, port, dir)
	if server == nil {
		fmt.Println("create topid chart server failed!")
		return
	}
	fmt.Println("topid chart server starting...")
	server.Start()

	return
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
