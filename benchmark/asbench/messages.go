package asbench

import (
	as "github.com/godevsig/adaptiveservice"
)

const (
	// Publisher is the service(s) publisher
	Publisher = "benchmark"
	// Service is the asbench service
	Service = "asbench"
)

// UploadRequest is the message with content to be uploaded.
// Return 0 or error.
type UploadRequest struct {
	Name    string
	ID      int32
	Payload []byte
}

// DownloadRequest is the message that asks content with Specified Name and ID.
// Return []byte or error.
type DownloadRequest struct {
	Name string
	ID   int32
	Size int32 // in byte
}

func init() {
	as.RegisterType((*UploadRequest)(nil))
	as.RegisterType((*DownloadRequest)(nil))
}
