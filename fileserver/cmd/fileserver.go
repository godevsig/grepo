package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/godevsig/glib/sys/log"
	"github.com/godevsig/grepo/fileserver"
)

var fs *fileserver.FileServer

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	logLevel := flags.String("logLevel", "info", "debug/info/warn/error")
	dir := flags.String("dir", "", "absolute directory path to be served")
	port := flags.String("port", "0", "set server port, default 0 means alloced by net Listener")
	title := flags.String("title", "file server", "title of file server")
	if err = flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	if len(*dir) == 0 {
		return fmt.Errorf("no dir specified")
	}

	stream := log.NewStream("")
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("fileserver", log.StringToLoglevel(*logLevel))

	fs = fileserver.NewFileServer(lg, *port, *dir, *title)
	if fs == nil {
		return errors.New("create file server failed")
	}

	fmt.Printf("file server for %s @ :%s\n", *dir, fs.Port)

	return fs.Start()
}

// Stop stops the app
func Stop() {
	fmt.Println("file server stopping...")
	fs.Stop()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
