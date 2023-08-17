package client

import (
	"errors"
	"fmt"
	"net"
	"time"

	asbench "github.com/godevsig/grepo/asbench/raw/server"
)

var content []byte

func init() {
	content = make([]byte, 1<<18) //256K bytes
	for i := range content {
		content[i] = 5
	}
}

// Run runs the app
func Run(transport, test string, size, duration int, packet bool) (err error) {
	var netconn net.Conn
	switch transport {
	case "uds":
		netconn, err = net.Dial("unix", "asbenchrawserver.sock")
	case "tcp":
		netconn, err = net.Dial("tcp", "127.0.0.1:9588")
	}
	if err != nil {
		return err
	}
	defer netconn.Close()

	mio := asbench.NewMessageIO(netconn)

	read := mio.ReadStream
	write := mio.WriteStream
	if packet {
		read = mio.ReadPacket
		write = mio.WritePacket
	}

	running := true

	dld := func() (counter int64, err error) {
		req := asbench.DownloadRequest{Name: "testdld", ID: int32(1), Size: int32(size)}
		fmt.Println("request:", req)
		var msg interface{}
		for running {
			if err = write(req); err != nil {
				return
			}
			msg, err = read()
			if err != nil {
				return
			}
			counter++
		}
		fmt.Println("reply:", msg)
		return
	}

	uld := func() (counter int64, err error) {
		req := asbench.UploadRequest{Name: "testuld", ID: int32(2), Payload: content[:size]}
		fmt.Println("request:", req)
		var msg interface{}
		for running {
			if err = write(req); err != nil {
				return
			}
			msg, err = read()
			if err != nil {
				return
			}
			counter++
		}
		fmt.Println("reply:", msg)
		return
	}

	var bench func() (int64, error)
	switch test {
	case "download":
		bench = dld
	case "upload":
		bench = uld
	default:
		return errors.New("unknown test type")
	}

	go func() { time.Sleep(time.Duration(duration) * time.Second); running = false }()

	//runtime.LockOSThread()
	start := time.Now()
	total, err := bench()
	end := time.Now()
	//runtime.UnlockOSThread()
	elapsed := end.Sub(start).Seconds()

	if err != nil {
		return err
	}
	fmt.Println(start, end)
	fmt.Printf("Transaction Per Second(TPS): %12.02f\n", float64(total)/elapsed)
	return nil
}
