package main

import (
	"fmt"
	"runtime"
)

func main() {
	buf := make([]byte, 4096)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	fmt.Printf("%s\n", buf)
}
