package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/niubaoshu/gotiny"
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

func registerType(i interface{}) {
	//gotiny.Register(i)
	rt := reflect.TypeOf(i)
	gotiny.RegisterName(rt.String(), rt)
}

func init() {
	registerType([]byte(nil))
	registerType(UploadRequest{})
	registerType(DownloadRequest{})
}

var content []byte

func init() {
	content = make([]byte, 1<<18) //256K bytes
	for i := range content {
		content[i] = 9
	}
}

// MessageIO is message reader and writer on netconn.
type MessageIO struct {
	netconn net.Conn
	bufSize []byte
	bufMsg  []byte
	enc     *gotiny.Encoder
	dec     *gotiny.Decoder
}

// NewMessageIO creates a MessageIO.
func NewMessageIO(netconn net.Conn) *MessageIO {
	var msg interface{}
	mio := &MessageIO{
		netconn: netconn,
		bufSize: make([]byte, 4),
		bufMsg:  make([]byte, 1<<16+1),
		enc:     gotiny.NewEncoderWithPtr(&msg),
		dec:     gotiny.NewDecoderWithPtr(&msg),
	}
	mio.dec.SetCopyMode()
	return mio
}

// ReadStream reads a message from MessageIO in stream mode.
func (mio *MessageIO) ReadStream() (msg interface{}, err error) {
	if _, err = io.ReadFull(mio.netconn, mio.bufSize); err != nil {
		return
	}
	size := binary.BigEndian.Uint32(mio.bufSize)
	bufCap := uint32(cap(mio.bufMsg))
	if size <= bufCap {
		mio.bufMsg = mio.bufMsg[:size]
	} else {
		mio.bufMsg = make([]byte, size)
	}
	if _, err = io.ReadFull(mio.netconn, mio.bufMsg); err != nil {
		return
	}
	mio.dec.Decode(mio.bufMsg, &msg)
	return
}

// WriteStream writes a message to MessageIO in stream mode.
func (mio *MessageIO) WriteStream(msg interface{}) (err error) {
	buf := net.Buffers{}
	bufMsg := mio.enc.Encode(&msg)
	bufSize := make([]byte, 4)
	binary.BigEndian.PutUint32(bufSize, uint32(len(bufMsg)))
	buf = append(buf, bufSize, bufMsg)
	_, err = buf.WriteTo(mio.netconn)
	return
}

// ReadPacket reads a message from MessageIO in packet mode.
func (mio *MessageIO) ReadPacket() (msg interface{}, err error) {
	n, err := mio.netconn.Read(mio.bufMsg)
	if err != nil {
		return nil, err
	}
	/*
		if n != 34 && n != 42 {
			fmt.Println(n, string(mio.bufMsg[:n]))
			panic(n)
		}
	*/
	mio.dec.Decode(mio.bufMsg[:n], &msg)
	return
}

// WritePacket writes a message to MessageIO in packet mode.
func (mio *MessageIO) WritePacket(msg interface{}) (err error) {
	bufMsg := mio.enc.Encode(&msg)
	_, err = mio.netconn.Write(bufMsg)
	return
}

// Run runs server
func Run(packet bool) error {
	unixLnr, err := net.Listen("unix", "asbenchrawserver.sock")
	if err != nil {
		return err
	}

	tcpLnr, err := net.Listen("tcp", ":9588")
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)

	handleConn := func(netconn net.Conn) {
		defer netconn.Close()
		mio := NewMessageIO(netconn)

		read := mio.ReadStream
		write := mio.WriteStream
		if packet {
			read = mio.ReadPacket
			write = mio.WritePacket
		}

		for {
			msg, err := read()
			if err != nil {
				errChan <- err
				return
			}

			switch msg := msg.(type) {
			case UploadRequest:
				if err := write(0); err != nil {
					errChan <- err
					return
				}
			case DownloadRequest:
				if err := write(content[:msg.Size]); err != nil {
					errChan <- err
					return
				}
			}
		}
	}

	for _, lnr := range []net.Listener{unixLnr, tcpLnr} {
		go func(lnr net.Listener) {
			for {
				netconn, err := lnr.Accept()
				if err != nil {
					errChan <- err
					return
				}
				go handleConn(netconn)
			}
		}(lnr)
	}

	for {
		err := <-errChan
		if err != io.EOF {
			fmt.Println(err)
			break
		}
	}
	return nil
}
