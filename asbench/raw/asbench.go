package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/godevsig/grepo/asbench/raw/client"
	"github.com/godevsig/grepo/asbench/raw/server"
)

func main() {
	packet := flag.Bool("packet", false, "packet or stream")
	mode := flag.String("mode", "cs", "cs or c or s")
	transport := flag.String("transport", "uds", "uds or tcp")
	test := flag.String("type", "download", "test type: download or upload")
	size := flag.Int("s", 32, "payload size in byte")
	tm := flag.Int("t", 3, "test for how long")

	flag.Parse()

	switch *mode {
	case "c":
		if err := client.Run(*transport, *test, *size, *tm, *packet); err != nil {
			fmt.Println(err)
		}
	case "s":
		if err := server.Run(*packet); err != nil {
			fmt.Println(err)
		}
	default:
		go func() {
			time.Sleep(time.Second)
			if err := client.Run(*transport, *test, *size, *tm, *packet); err != nil {
				fmt.Println(err)
			}
		}()
		if err := server.Run(*packet); err != nil {
			fmt.Println(err)
		}
	}

	return
}
