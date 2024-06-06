//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type wsConn struct {
	conn     net.Conn
	isServer bool
}

func NewWSConn(conn net.Conn, isServer bool) *wsConn {
	return &wsConn{conn: conn, isServer: isServer}
}

func (c *wsConn) Read(b []byte) (n int, err error) {
	header, err := ws.ReadHeader(c.conn)
	if err != nil {
		return 0, err
	}
	buf := make([]byte, header.Length)
	payload := buf[:header.Length]
	_, err = io.ReadFull(c.conn, payload)
	if err != nil {
		return 0, err
	}
	if header.Masked {
		ws.Cipher(payload, header.Mask, 0)
	}
	if len(payload) > len(b) {
		return 0, fmt.Errorf("buffer size:%d too small to transport ws payload size:%d", len(b), len(payload))
	}
	copy(b, payload)
	return len(payload), nil
}

func (c *wsConn) Write(b []byte) (n int, err error) {
	if c.isServer {
		err = wsutil.WriteServerBinary(c.conn, b)
	} else {
		err = wsutil.WriteClientBinary(c.conn, b)
	}
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *wsConn) Close() error {
	return c.conn.Close()
}

func HandleRequest(w http.ResponseWriter, req *http.Request) {
	wsc, _, _, err := ws.UpgradeHTTP(req, w)
	if err != nil {
		return
	}
	rc, err := net.Dial("tcp", "127.0.0.1")
	if err != nil {
		println("err,", err.Error())
	}
	io.Copy(NewWSConn(wsc, false), NewWSConn(rc, true))
	wsc.Close()
	rc.Close()
}

func main() {
	fmt.Println("Hello Web Assembly from Go!")

	js.Global().Set("setHtml", SetHtml()) // 带参函数
}

func SetHtml() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			text := args[0].String()
			return strings.Replace("pwd", "{{ placeholder }}", text, -1)
		}
		return js.Undefined()
	})
}
