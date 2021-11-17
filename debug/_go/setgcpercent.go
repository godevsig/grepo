package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s percent\n", os.Args[0])
		return
	}
	percent, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(debug.SetGCPercent(percent))
}
