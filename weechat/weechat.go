// package weechat implements the WeeChat relay protocol.
package weechat

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"net"
)

// Reference: http://www.weechat.org/files/doc/stable/weechat_relay_protocol.en.html

type command int

const (
	cmdInit command = iota
	cmdHdata
	cmdInfo
	cmdInfolist
	cmdNicklist
	cmdInput
	cmdSync
	cmdDesync
	cmdQuit
	cmdCount
)

var cmdStrings = [cmdCount]string{
	cmdInit:     "init",
	cmdHdata:    "hdata",
	cmdInfo:     "info",
	cmdInfolist: "infolist",
	cmdNicklist: "nicklist",
	cmdInput:    "input",
	cmdSync:     "sync",
	cmdDesync:   "desync",
	cmdQuit:     "quit",
}

type Conn struct {
	c net.Conn
	r *bufio.Reader
}

func Dial(addr string) (*Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Conn{c: conn, r: bufio.NewReader(conn)}, nil
}

func (conn *Conn) send(cmd command, args ...string) error {
	buf := make([]byte, 0, 80)
	buf = append(buf, cmdStrings[cmd]...)
	for _, a := range args {
		buf = append(buf, ' ')
		buf = append(buf, a...)
	}
	buf = append(buf, '\n')
	_, err := conn.c.Write(buf)
	return err
}

var errMsgTooLarge = errors.New("message too large")

// recv gets a message from the connection.
func (conn *Conn) recv() (s []byte, err error) {
	// A message is:
	// - a uint32 length
	// - a byte boolean for compression
	// - length-5 bytes of data (plain or zlib compressed)
	var buf [5]byte
	_, err = io.ReadFull(conn.r, buf[:])
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(buf[:4])
	isCompressed := buf[4] == 1
	if length >= 32<<20 {
		return nil, errMsgTooLarge
	}

	s = make([]byte, length-5)
	_, err = io.ReadFull(conn.r, s)
	if err != nil {
		return
	}
	if isCompressed {
		zr, err := zlib.NewReader(bytes.NewBuffer(s))
		if err != nil {
			return s, err
		}
		return ioutil.ReadAll(zr)
	}
	return s, nil
}

func (conn *Conn) ListBuffers() ([]Buffer, error) {
	var s []byte
	err := conn.send(cmdHdata, "buffer:gui_buffers(*)")
	if err == nil {
		s, err = conn.recv()
	}
	if err != nil {
		return nil, err
	}
	msg := message(s)
	id, typ := msg.Buffer(), msg.GetType()
	//t.Logf("id=%s type=%v", id, typ)
	_, _ = id, typ
	var buflist []Buffer
	msg.HData(&buflist)
	return buflist, nil
}
