// Code generated by original messages.go. DO NOT EDIT.

package recorder

import (
	as "github.com/godevsig/adaptiveservice"
)

// SessionRequest is the message sent by client.
// Return SessionResponse.
// Client should send one or more Record after SessionResponse is received.
type SessionRequest struct {
	Tag string
}

// SessionResponse is the message replied by server.
type SessionResponse struct {
	RecorderURL string
}

func init() {
	as.RegisterType((*SessionRequest)(nil))
	as.RegisterType((*SessionResponse)(nil))
}
