package main

import "runtime/debug"

func main() {
	debug.FreeOSMemory()
}
