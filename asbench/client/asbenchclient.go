package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/asbench"
	"github.com/godevsig/glib/sys/log"
)

var content []byte

func init() {
	content = make([]byte, 1<<18) //256K bytes
	for i := range content {
		content[i] = 5
	}
}

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	logLevel := flags.String("logLevel", "info", "debug/info/warn/error")
	scope := flags.String("scope", "os", "process/os/lan/wan")
	test := flags.String("type", "download", "test type: download or upload")
	number := flags.Int("n", 1, "parallel number")
	size := flags.Int("s", 32, "payload size in byte")
	tm := flags.Int("t", 3, "test for how long")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	stream := log.NewStream("")
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("asbenchc", log.StringToLoglevel(*logLevel))

	Scope := as.ScopeOS
	switch *scope {
	case "process":
		Scope = as.ScopeProcess
	case "os":
		Scope = as.ScopeOS
	case "lan":
		Scope = as.ScopeLAN
	case "wan":
		Scope = as.ScopeWAN
	default:
		return errors.New("wrong scope")
	}
	c := as.NewClient(as.WithLogger(lg), as.WithScope(Scope)).SetDiscoverTimeout(3)
	conn := <-c.Discover(asbench.Publisher, asbench.Service)
	if conn == nil {
		return as.ErrServiceNotFound(asbench.Publisher, asbench.Service)
	}
	defer conn.Close()

	var running = true

	dld := func() int64 {
		stream := conn.NewStream()
		req := asbench.DownloadRequest{Name: "testdld", ID: int32(1), Size: int32(*size)}
		//fmt.Println("request:", req)
		var rep []byte
		var counter int64
		for running {
			if err := stream.SendRecv(&req, &rep); err != nil {
				panic(err)
			}
			counter++
		}
		//fmt.Println("reply:", rep)
		return counter
	}

	uld := func() int64 {
		stream := conn.NewStream()
		req := asbench.UploadRequest{Name: "testuld", ID: int32(2), Payload: content[:*size]}
		//fmt.Println("request:", req)
		var counter int64
		var rep int
		for running {
			if err := stream.SendRecv(&req, &rep); err != nil {
				panic(err)
			}
			counter++
		}
		//fmt.Println("reply:", rep)
		return counter
	}

	var bench func() int64
	switch *test {
	case "download":
		bench = dld
	case "upload":
		bench = uld
	default:
		return errors.New("unknown test type")
	}

	counters := make(chan int64)
	go func() { time.Sleep(time.Duration(*tm) * time.Second); running = false }()

	start := time.Now()
	for n := 0; n < *number; n++ {
		go func() { counters <- bench() }()
	}

	var total int64
	for n := 0; n < *number; n++ {
		total += <-counters
	}
	elapsed := time.Now().Sub(start).Seconds()

	fmt.Printf("Transaction Per Second(TPS): %12.02f\n", float64(total)/elapsed)

	return nil
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
