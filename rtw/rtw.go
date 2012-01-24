package rtw

import (
	"fmt"
	"io"
	"websocket"
)

type ClientId interface{}

type MessageData []byte

type Message struct {
	Id  ClientId
	Message MessageData
}

type Handler interface {
	Connect(ws *websocket.Conn) ClientId
	Disconnect(id ClientId)
	Message(msg Message)
}

type clientInfo struct {
	id       ClientId
	toClient chan Message
	isActive bool
}

type RTW struct {
	fromClient chan Message
	clientCtrl chan clientInfo
	clients    map[ClientId]clientInfo
	handler    Handler
}

func NewRTW(h Handler) *RTW {
	rtw := &RTW{
		fromClient: make(chan Message),
		clientCtrl: make(chan clientInfo),
		clients:    make(map[ClientId]clientInfo),
		handler:    h,
	}
	return rtw
}

func (rtw *RTW) Run() {
	go rtw.manageClients()
}

func (rtw *RTW) Send(id ClientId, msg Message) {
	if ci, ok := rtw.clients[id]; ok {
		ci.toClient <- msg
	}
}

func (rtw *RTW) WSHandler() func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		id := rtw.handler.Connect(ws)
		toClient := make(chan Message)
		ci := clientInfo{id, toClient, true}
		rtw.clientCtrl <- ci
		go rtw.writeHandler(ws, ci)
		rtw.readHandler(ws, ci)
		ci.isActive = false
		rtw.handler.Disconnect(id)
		rtw.clientCtrl <- ci
	}
}

func (rtw *RTW) manageClients() {
	for {
		select {
		case msg := <-rtw.fromClient:
			rtw.handler.Message(msg)
		case ci := <-rtw.clientCtrl:
			if ci.isActive {
				rtw.clients[ci.id] = ci
			} else {
				close(ci.toClient)
				delete(rtw.clients, ci.id)
			}
		}
	}
}

func (rtw *RTW) readHandler(ws *websocket.Conn, ci clientInfo) {
	msg := make(MessageData, 512) // XXX can size be dynamic ???
	for {
		n, err := ws.Read(msg)
		switch err {
		case nil:
			rtw.fromClient <- Message{ci.id, msg[:n]}
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

func (rtw *RTW) writeHandler(ws *websocket.Conn, ci clientInfo) {
	for {
		msg, ok := <-ci.toClient
		if !ok {
			return
		}
		if _, err := ws.Write(msg.Message); err != nil {
			fmt.Println(err)
		}
	}
}
