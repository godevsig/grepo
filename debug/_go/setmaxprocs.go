package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s number\n", os.Args[0])
		return
	}
	n, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(runtime.GOMAXPROCS(n))
}
