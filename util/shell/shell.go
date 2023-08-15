package main

import (
	"fmt"
	"os"

	"github.com/godevsig/glib/sys/shell"
)

// Start starts the app
func Start(args []string) (err error) {
	return shell.RunWith(os.Stdin, os.Stdout)
}

// Stop stops the app
func Stop() {
	fmt.Println("use exit to quit")
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
