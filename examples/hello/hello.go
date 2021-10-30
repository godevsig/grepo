package main

import (
	"fmt"
	"os"
)

// Start starts the app
func Start(args []string) (err error) {
	fmt.Println("Hello, world!")
	return nil
}

// Stop stops the app
func Stop() {
	fmt.Println("stopping...")
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
