package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/recorder"
)

var (
	file string
	tag  string
	cmd  *exec.Cmd
)

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	flags.StringVar(&tag, "tag", "temp", "tag is part of the URL, used to mark this run")
	flags.StringVar(&file, "file", "", "file to be recorded")

	if err = flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	var conn as.Connection
	c := as.NewClient().SetDiscoverTimeout(3)
	conn = <-c.Discover("platform", "recorder")
	if conn == nil {
		return errors.New("connect to recorder failed")
	}
	defer conn.Close()

	sessionReq := recorder.SessionRequest{
		Tag: tag,
	}
	var sessionRep recorder.SessionResponse
	if err := conn.SendRecv(&sessionReq, &sessionRep); err != nil {
		return err
	}

	fmt.Println("Visit below URL to get the log:")
	fmt.Println(sessionRep.RecorderURL)

	cmd = exec.Command("tail", "-F", file)
	cmd.Stdout = as.NewStreamIO(conn)
	err = cmd.Run()
	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

// Stop stops the app
func Stop() {
	fmt.Println("recorder client stopping...")
	if err := cmd.Process.Kill(); err != nil {
		fmt.Println("failed to stop recorder client")
	}
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
