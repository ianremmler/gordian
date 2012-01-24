package gordian

import (
	"fmt"
	"io"
	"websocket"
)

const (
	MaxMessageSize = 1024
)

type ClientId interface{}

type MessageData []byte

type Message struct {
	Id      ClientId
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
	isAlive  bool
}

type Gordian struct {
	fromClient chan Message
	clientCtrl chan clientInfo
	clients    map[ClientId]clientInfo
	handler    Handler
}

func NewGordian(h Handler) *Gordian {
	g := &Gordian{
		fromClient: make(chan Message),
		clientCtrl: make(chan clientInfo),
		clients:    make(map[ClientId]clientInfo),
		handler:    h,
	}
	return g
}

func (g *Gordian) Run() {
	go g.manageClients()
}

func (g *Gordian) Send(id ClientId, msg Message) {
	if ci, ok := g.clients[id]; ok {
		ci.toClient <- msg
	}
}

func (g *Gordian) WSHandler() func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		id := g.handler.Connect(ws)
		if id == nil {
			return
		}
		toClient := make(chan Message)
		ci := clientInfo{id, toClient, true}
		g.clientCtrl <- ci
		go g.writeToWS(ws, ci)
		g.readFromWS(ws, ci)
		g.handler.Disconnect(id)
		ci.isAlive = false
		g.clientCtrl <- ci
	}
}

func (g *Gordian) manageClients() {
	for {
		select {
		case msg := <-g.fromClient:
			g.handler.Message(msg)
		case ci := <-g.clientCtrl:
			if ci.isAlive {
				g.clients[ci.id] = ci
			} else {
				close(ci.toClient)
				delete(g.clients, ci.id)
			}
		}
	}
}

func (g *Gordian) readFromWS(ws *websocket.Conn, ci clientInfo) {
	msg := make(MessageData, MaxMessageSize)
	for {
		n, err := ws.Read(msg)
		switch err {
		case nil:
			g.fromClient <- Message{ci.id, msg[:n]}
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

func (g *Gordian) writeToWS(ws *websocket.Conn, ci clientInfo) {
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
