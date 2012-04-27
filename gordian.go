// The gordian package provides a simple framework for building multiclient
// websocket applications.
package gordian

import (
	"code.google.com/p/go.net/websocket"

	"fmt"
	"io"
)

// Message is used to transfer messages between the application and clients.
type Message struct {
	// Id identifies the client that sent the message.
	Id string
	// Data contains the message payload.
	Data interface{}
}

// Handler defines the interface used by gordian to interact with the
// application when connection events occur.
type Handler interface {
	// Connect is called when a new websocket connection is initiated.
	// The application must generate a unique id for each client.
	// If an error is returned, the connection is dropped.
	Connect(ws *websocket.Conn) (string, error)
	// Disconnect is called when a websocket connection ends.
	Disconnect(id string)
	// HandleMessage is called to handle incoming messages from clients.
	HandleMessage(msg *Message)
}

// clientInfo conveys information about a client's state to clientManager.
type clientInfo struct {
	// id is the identifier supplied by the application for this client.
	id string
	// toClient is used to deliver messages to the client.
	toClient chan *Message
	// isAlive indicates whether the client just connected or disconnected.
	isAlive bool
}

// The Gordian class is responsible for managing client connections and
// reading and distributing client messages.
type Gordian struct {
	// fromClient sends messages from clients to gordian.
	fromClient chan *Message
	// clientCtrl sends client connection events and messages to
	// clientManager.
	clientCtrl chan *clientInfo
	// clients maps an id to a client's information.
	clients map[string]clientInfo
	// Handler is an object supplied by the application that handles
	// events and message passed to it by gordian.
	handler Handler
}

// New constructs a Gordian object.
func New(h Handler) *Gordian {
	g := &Gordian{
		fromClient: make(chan *Message),
		clientCtrl: make(chan *clientInfo),
		clients:    make(map[string]clientInfo),
		handler:    h,
	}
	return g
}

// Run starts the goroutine that manages connections and message distribution.
func (g *Gordian) Run() {
	go g.manageClients()
}

// Send passes a message to the specified client.
func (g *Gordian) Send(id string, msg *Message) {
	if ci, ok := g.clients[id]; ok {
		ci.toClient <- msg
	}
}

// WSHandler returns a function to be called by http.ListenAndServe to handle
// a new websocket connection.
func (g *Gordian) WSHandler() func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		id, err := g.handler.Connect(ws)
		if err != nil {
			return
		}
		toClient := make(chan *Message)
		ci := &clientInfo{id, toClient, true}
		g.clientCtrl <- ci
		go g.writeToWS(ws, ci)
		g.readFromWS(ws, ci)
		g.handler.Disconnect(id)
		ci.isAlive = false
		g.clientCtrl <- ci
	}
}

// manageClients waits for connection or message events and updates the
// internal state or delivers the message, respectively.
func (g *Gordian) manageClients() {
	for {
		select {
		case msg := <-g.fromClient:
			g.handler.HandleMessage(msg)
		case ci := <-g.clientCtrl:
			if ci.isAlive {
				g.clients[ci.id] = *ci
			} else {
				close(ci.toClient)
				delete(g.clients, ci.id)
			}
		}
	}
}

// readFromWS reads a message from the client and passes it to clientManager.
func (g *Gordian) readFromWS(ws *websocket.Conn, ci *clientInfo) {
	for {
		var data interface{}
		err := websocket.JSON.Receive(ws, &data)
		switch err {
		case nil:
			g.fromClient <- &Message{ci.id, data}
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

// writeToWS gets messages from clientManager and sends them to the client.
func (g *Gordian) writeToWS(ws *websocket.Conn, ci *clientInfo) {
	for {
		msg, ok := <-ci.toClient
		if !ok {
			return
		}
		if err := websocket.JSON.Send(ws, msg.Data); err != nil {
			fmt.Println(err)
		}
	}
}
