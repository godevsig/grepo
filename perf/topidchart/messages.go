package topidchart

import (
	as "github.com/godevsig/adaptiveservice"
)

// SessionRequest is the message sent by client.
// Return SessionResponse.
// Client should send one or more Record after SessionResponse is received.
type SessionRequest struct {
	Tag       string
	SysInfo   SysInfo
	ExtraInfo string
}

// SessionResponse is the message replied by server.
type SessionResponse struct {
	ChartURL string
}

// ProcessInfo is process statistics.
type ProcessInfo struct {
	Pid  int
	Name string
	Ucpu float32
	Scpu float32
	Mem  uint64 // in KB
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

func init() {
	as.RegisterType((*SessionRequest)(nil))
	as.RegisterType((*SessionResponse)(nil))
	as.RegisterType((*Record)(nil))
}

//go:generate mkdir -p $GOPACKAGE
//go:generate sh -c "grep -v go:generate $GOFILE > $GOPACKAGE/$GOFILE"
//go:generate gopls format -w $GOPACKAGE/$GOFILE
//go:generate git add $GOPACKAGE/$GOFILE
