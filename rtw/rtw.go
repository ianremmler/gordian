package rtw

import (
	"fmt"
	"io"
	"websocket"
)

type clientInfo struct {
	input    chan []byte
	isActive bool
}

type RTW struct {
	Handler    func(ws *websocket.Conn)
	output     chan []byte
	clientCtrl chan clientInfo
	clients    map[chan []byte]struct{}
}

func NewRTW() *RTW {
	rtw := &RTW{
		output:     make(chan []byte),
		clientCtrl: make(chan clientInfo),
		clients:    make(map[chan []byte]struct{}),
	}
	rtw.Handler = rtw.wsHandler()
	return rtw
}

func (rtw *RTW) Run() {
	go rtw.manageClients()
}

func (rtw *RTW) wsHandler() func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		input := make(chan []byte)
		rtw.clientCtrl <- clientInfo{input, true}
		go rtw.writeHandler(ws, input)
		rtw.readHandler(ws)
		rtw.clientCtrl <- clientInfo{input, false}
	}
}

func (rtw *RTW) manageClients() {
	for {
		select {
		case msg := <-rtw.output:
			for input, _ := range rtw.clients {
				input <- msg
			}
		case ci := <-rtw.clientCtrl:
			if ci.isActive {
				rtw.clients[ci.input] = struct{}{}
			} else {
				close(ci.input)
				delete(rtw.clients, ci.input)
			}
		}
	}
}

func (rtw *RTW) readHandler(ws *websocket.Conn) {
	msg := make([]byte, 512)
	for {
		n, err := ws.Read(msg)
		switch err {
		case nil:
			rtw.output <- msg[:n]
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

func (rtw *RTW) writeHandler(ws *websocket.Conn, input chan []byte) {
	for {
		msg, ok := <-input
		if !ok {
			return
		}
		if _, err := ws.Write(msg); err != nil {
			fmt.Println(err)
		}
	}
}
