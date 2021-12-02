package asbench

import (
	as "github.com/godevsig/adaptiveservice"
)

var content []byte

func init() {
	content = make([]byte, 1<<18) //256K bytes
	for i := range content {
		content[i] = 9
	}
}

// Handle handles msg.
func (msg UploadRequest) Handle(stream as.ContextStream) (reply interface{}) {
	// discard everything, reply 0 to client.
	return 0
}

// Handle handles msg.
func (msg DownloadRequest) Handle(stream as.ContextStream) (reply interface{}) {
	// reply fixed value content to client.
	return content[:msg.Size]
}

var knownMsgs = []as.KnownMessage{
	UploadRequest{},
	DownloadRequest{},
}
