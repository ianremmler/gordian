package gordian

import (
	"code.google.com/p/go.net/websocket"

	"fmt"
	"io"
)

const (
	CONNECT = iota
	DISCONNECT
)

// ClientId identifies a websocket client.  It must be usable as a map key
type ClientId interface{}

// MessageData is the unmarshalled payload of an websocket message
type MessageData interface{}

// Message is used to transfer messages between the application and clients.
type Message struct {
	// From identifies the client that sent the message.
	From ClientId
	// To identifies the client that receives the message.
	To ClientId
	// Data contains the message payload.
	Data MessageData
}

// ClientData holds information about a client's state
type ClientData struct {
	Id       ClientId
	CtrlType int
	toClient chan *Message
}

// The Gordian class is responsible for managing client connections and reading
// and distributing client messages.
type Gordian struct {
	Connect    chan *websocket.Conn
	Control    chan *ClientData
	Messages   chan *Message
	clientCtrl chan *ClientData
	clients    map[ClientId]ClientData
}

// New constructs a Gordian object.
func New() *Gordian {
	g := &Gordian{
		Connect:    make(chan *websocket.Conn),
		Control:    make(chan *ClientData),
		Messages:   make(chan *Message),
		clientCtrl: make(chan *ClientData),
		clients:    make(map[ClientId]ClientData),
	}
	return g
}

// Run starts the goroutine that manages connections and message distribution.
func (g *Gordian) Run() {
	go g.manageClients()
}

// WSHandler returns a function to be called by http.ListenAndServe to handle a
// new websocket connection.
func (g *Gordian) WSHandler() func(conn *websocket.Conn) {
	return func(conn *websocket.Conn) {
		g.Connect <- conn
		clientData := <-g.Control
		if clientData.Id == nil || clientData.CtrlType != CONNECT {
			return
		}
		clientData.toClient = make(chan *Message)
		g.clientCtrl <- clientData
		go g.writeToWS(conn, clientData)
		g.readFromWS(conn, clientData)
		clientData.CtrlType = DISCONNECT
		g.Control <- clientData
		clientData.CtrlType = DISCONNECT
		g.clientCtrl <- clientData
	}
}

// manageClients waits for connection or message events and updates the
// internal state or delivers the message, respectively.
func (g *Gordian) manageClients() {
	for {
		select {
		case msg := <-g.Messages:
			if clientData, ok := g.clients[msg.To]; ok {
				clientData.toClient <- msg
			}
		case clientData := <-g.clientCtrl:
			if clientData.CtrlType == CONNECT {
				g.clients[clientData.Id] = *clientData
			} else {
				close(clientData.toClient)
				delete(g.clients, clientData.Id)
			}
		}
	}
}

// readFromWS reads a message from the client and passes it to clientManager.
func (g *Gordian) readFromWS(conn *websocket.Conn, clientData *ClientData) {
	for {
		var data MessageData
		err := websocket.JSON.Receive(conn, &data)
		switch err {
		case nil:
			g.Messages <- &Message{clientData.Id, nil, data}
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

// writeToWS gets messages from clientManager and sends them to the client.
func (g *Gordian) writeToWS(conn *websocket.Conn, clientData *ClientData) {
	for {
		msg, ok := <-clientData.toClient
		if !ok {
			return
		}
		if err := websocket.JSON.Send(conn, msg.Data); err != nil {
			fmt.Println(err)
		}
	}
}
