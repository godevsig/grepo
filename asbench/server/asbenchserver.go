package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/godevsig/grepo/asbench"
	"github.com/godevsig/glib/sys/log"
)

var server *asbench.Server

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
	lg := stream.NewLogger("asbenchs", log.StringToLoglevel(*logLevel))

	fmt.Println("asbench server starting...")
	server = asbench.NewServer(lg)
	if server == nil {
		return errors.New("create asbench server failed")
	}

	return server.Run()
}

// Stop stops the app
func Stop() {
	fmt.Println("asbench server stopping...")
	server.Stop()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
