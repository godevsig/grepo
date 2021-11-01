package topid

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	as "github.com/godevsig/adaptiveservice"
)

type pRecord struct {
	Timestamp int64
	Processes []ProcessInfo
}

type sRecord struct {
	Timestamp int64
	Snapshot  string
}

// ProcessInfo is process statistics.
type ProcessInfo struct {
	Pid  int
	Name string
	Ucpu float64
	Scpu float64
	Mem  uint64
}

// Record is sent by client periodically including target processes info,
// an optional snapshot such as process tree, and timestamp.
type Record struct {
	Timestamp int64
	Processes []ProcessInfo
	Snapshot  string
}

// SysInfo is part of SessionRequest used to initiate a collecting session.
type SysInfo struct {
	CPUInfo    string
	KernelInfo string
}

// SessionRequest is the message sent by client.
// Return SessionResponse.
// Client should send one or more Record after SessionResponse is received.
type SessionRequest struct {
	Tag       string
	SysInfo   SysInfo
	ExtraInfo string
}

// Handle handles SessionRequest.
func (msg *SessionRequest) Handle(stream as.ContextStream) (reply interface{}) {
	id := time.Now().Format("20060102") + "-" + randStringRunes(8)
	info = fmt.Sprintf("------CPUInfo------\n%s\n------KernelInfo------\n%s\n------ExtraInfo------\n%s\n", msg.SysInfo.CPUInfo, msg.SysInfo.KernelInfo, msg.ExtraInfo)

	go func() {
		var buf = &Record{}
		var pbuf = &pRecord{}
		var sbuf = &sRecord{}
		filepath := fmt.Sprintf("%v/%v", cfg.dir, msg.Tag)
		process := fmt.Sprintf("process-%v.data", id)
		snapshot := fmt.Sprintf("snapshot-%v.data", id)
		err := os.MkdirAll(filepath, 0777)
		if err != nil {
			panic(err)
		}
		processFile, err := os.OpenFile(path.Join(filepath, process), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		defer processFile.Close()
		if err != nil {
			panic(err)
		}
		snapshotFile, err := os.OpenFile(path.Join(filepath, snapshot), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		defer snapshotFile.Close()
		if err != nil {
			panic(err)
		}
		pEnc := gob.NewEncoder(processFile)
		sEnc := gob.NewEncoder(snapshotFile)

		for {
			err := stream.Recv(buf)
			if err != nil {
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					fmt.Println(err)
				}
				break
			}

			pbuf.Timestamp = buf.Timestamp
			pbuf.Processes = buf.Processes
			pEnc.Encode(pbuf)

			if buf.Snapshot != "" {
				sbuf.Timestamp = buf.Timestamp
				sbuf.Snapshot = buf.Snapshot
				sEnc.Encode(sbuf)
			}
		}
	}()

	return &SessionResponse{fmt.Sprintf("http://%v:%v/%v/%v", cfg.ip, cfg.chartport, msg.Tag, id)}
}

// SessionResponse is the message replied by server.
type SessionResponse struct {
	ChartURL string
}

var knownMsgs = []as.KnownMessage{
	(*SessionRequest)(nil),
}

func init() {
	as.RegisterType((*SessionRequest)(nil))
	as.RegisterType((*SessionResponse)(nil))
	as.RegisterType((*Record)(nil))
}
