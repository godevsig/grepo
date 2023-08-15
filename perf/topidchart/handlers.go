package topidchart

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/glib/sys/log"
)

type pRecord struct {
	Timestamp int64
	Processes []ProcessInfo
}

type sRecord struct {
	Timestamp int64
	Snapshot  string
}

// Handle handles SessionRequest.
func (msg *SessionRequest) Handle(stream as.ContextStream) (reply interface{}) {
	lg := stream.GetContext().(*log.Logger)
	id := time.Now().Format("20060102") + "-" + randStringRunes(8)

	filepath := fmt.Sprintf("%v/%v", dataDir, msg.Tag)
	info := fmt.Sprintf("info-%v.data", id)
	process := fmt.Sprintf("process-%v.data", id)
	snapshot := fmt.Sprintf("snapshot-%v.data", id)
	if err := os.MkdirAll(filepath, 0777); err != nil {
		return err
	}

	infoFile, err := os.OpenFile(path.Join(filepath, info), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	infoFile.WriteString(fmt.Sprintf("------CPUInfo------\n%s\n------KernelInfo------\n%s\n------ExtraInfo------\n%s\n", msg.SysInfo.CPUInfo, msg.SysInfo.KernelInfo, msg.ExtraInfo))
	infoFile.Close()

	processFile, err := os.OpenFile(path.Join(filepath, process), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	snapshotFile, err := os.OpenFile(path.Join(filepath, snapshot), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	pEnc := gob.NewEncoder(processFile)
	sEnc := gob.NewEncoder(snapshotFile)

	go func() {
		defer func() { processFile.Close(); snapshotFile.Close() }()
		lg.Debugln("data processing started")

		for {
			var record Record
			err := stream.Recv(&record)
			if err != nil {
				if err != io.EOF && err != io.ErrUnexpectedEOF {
					lg.Errorln(err)
				}
				break
			}
			pEnc.Encode(&pRecord{record.Timestamp, record.Processes})

			if record.Snapshot != "" {
				sEnc.Encode(&sRecord{record.Timestamp, record.Snapshot})
			}
		}
	}()

	return &SessionResponse{fmt.Sprintf("http://%v/%v/%v", hostAddr, msg.Tag, id)}
}

var knownMsgs = []as.KnownMessage{
	(*SessionRequest)(nil),
}
