package gordian

import (
	"code.google.com/p/go.net/websocket"

	"fmt"
	"io"
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

type ClientInfo struct {
	Id      ClientId
	IsAlive bool
}

// clientData conveys information about a client's state to clientManager.
type clientData struct {
	// toClient is used to deliver messages to the client.
	toClient chan *Message

	ClientInfo
}

// The Gordian class is responsible for managing client connections and
// reading and distributing client messages.
type Gordian struct {
	Messages chan *Message
	Connect  chan *websocket.Conn
	Manage   chan *ClientInfo

	// clientCtrl sends client connection events and messages to
	// clientManager.
	clientCtrl chan *clientData
	// clients maps an id to a client's information.
	clients map[ClientId]clientData
}

// New constructs a Gordian object.
func New() *Gordian {
	g := &Gordian{
		Messages: make(chan *Message),
		Connect:  make(chan *websocket.Conn),
		Manage:   make(chan *ClientInfo),

		clientCtrl: make(chan *clientData),
		clients:    make(map[ClientId]clientData),
	}
	return g
}

// Run starts the goroutine that manages connections and message distribution.
func (g *Gordian) Run() {
	go g.manageClients()
}

// WSHandler returns a function to be called by http.ListenAndServe to handle
// a new websocket connection.
func (g *Gordian) WSHandler() func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		g.Connect <- ws
		ci := <-g.Manage
		if ci.Id == nil || !ci.IsAlive {
			return
		}
		toClient := make(chan *Message)
		cd := &clientData{toClient, *ci}
		g.clientCtrl <- cd
		go g.writeToWS(ws, cd)
		g.readFromWS(ws, cd)
		ci.IsAlive = false
		g.Manage <- ci
		cd.IsAlive = false
		g.clientCtrl <- cd
	}
}

// manageClients waits for connection or message events and updates the
// internal state or delivers the message, respectively.
func (g *Gordian) manageClients() {
	for {
		select {
		case msg := <-g.Messages:
			if cd, ok := g.clients[msg.To]; ok {
				cd.toClient <- msg
			}
		case cd := <-g.clientCtrl:
			if cd.IsAlive {
				g.clients[cd.Id] = *cd
			} else {
				close(cd.toClient)
				delete(g.clients, cd.Id)
			}
		}
	}
}

// readFromWS reads a message from the client and passes it to clientManager.
func (g *Gordian) readFromWS(ws *websocket.Conn, cd *clientData) {
	for {
		var data MessageData
		err := websocket.JSON.Receive(ws, &data)
		switch err {
		case nil:
			g.Messages <- &Message{cd.Id, nil, data}
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

// writeToWS gets messages from clientManager and sends them to the client.
func (g *Gordian) writeToWS(ws *websocket.Conn, cd *clientData) {
	for {
		msg, ok := <-cd.toClient
		if !ok {
			return
		}
		if err := websocket.JSON.Send(ws, msg.Data); err != nil {
			fmt.Println(err)
		}
	}
}
