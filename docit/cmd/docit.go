package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	_ "embed" //embed: read file

	"github.com/godevsig/glib/sys/log"
	"github.com/godevsig/grepo/docit"
)

var server *docit.Server

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
	lg := stream.NewLogger("docit", log.StringToLoglevel(*logLevel))

	fmt.Println("docit server starting...")
	server = docit.NewServer(lg)
	if server == nil {
		return errors.New("create docit server failed")
	}

	return server.Run()
}

// Stop stops the app
func Stop() {
	fmt.Println("docit server stopping...")
	server.Close()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
