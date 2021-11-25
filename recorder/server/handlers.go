package server

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	as "github.com/godevsig/adaptiveservice"
)

// Handle handles SessionRequest.
func (msg *SessionRequest) Handle(stream as.ContextStream) (reply interface{}) {
	id := time.Now().Format("20060102") + "-" + randStringRunes(8)

	filepath := fmt.Sprintf("%v/%v", dataDir, msg.Tag)
	log := fmt.Sprintf("%v.log", id)
	if err := os.MkdirAll(filepath, 0777); err != nil {
		return err
	}

	file, err := os.OpenFile(path.Join(filepath, log), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	go func() {
		buffer := as.NewStreamIO(stream)
		io.Copy(file, buffer)
	}()

	return &SessionResponse{fmt.Sprintf("http://%v/%v/%v", hostAddr, msg.Tag, log)}
}

var knownMsgs = []as.KnownMessage{
	(*SessionRequest)(nil),
}
