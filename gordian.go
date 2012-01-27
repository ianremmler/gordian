// The gordian package provides a simple framework for building multiclient
// websocket applications.
package gordian

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
)

const (
	// Default maximum message size is 64kB.
	DefaultMaxMsgSize = 1 << 16
)

// ClientId is used to identifiy gordian clients.  It can be any type that can
// be used as a map key.  A ClientId is supplied by the application when a new
// connection is established.
type ClientId interface{}

// MessageData is a buffer that contains message data.
type MessageData []byte

// Message is used to transfer messages between the application and clients.
type Message struct {
	// Id identifies the client that sent the message.
	Id ClientId
	// Message contains the data payload.
	Message MessageData
}

// Handler defines the interface used by gordian to interact with the
// application when connection events occur.
type Handler interface {
	// Connect is called when a new websocket connection is initiated.  The
	// application may use any method to generate a unique ClientId for
	// each client.
	Connect(ws *websocket.Conn) ClientId
	// Disconnect is called when a websocket connection ends.
	Disconnect(id ClientId)
	// Message is called to handle incoming messages from clients.
	Message(msg Message)
}

// clientInfo conveys information about a client's state to clientManager.
type clientInfo struct {
	// id is the identifier supplied by the application for this client.
	id ClientId
	// toClient is used to deliver messages to the client.
	toClient chan Message
	// isAlive indicates whether the client just connected or disconnected.
	isAlive bool
}

// The Gordian class is responsible for managing client connections and
// reading and distributing client messages.
type Gordian struct {
	// fromClient sends messages from clients to gordian.
	fromClient chan Message
	// clientCtrl sends client connection events and messages to clientManager.
	clientCtrl chan clientInfo
	// clients maps an id to a client's information.
	clients map[ClientId]clientInfo
	// handler is an object supplied by the application that implements the
	// Handler interface, used to events and message to the application.
	handler Handler
	// maxMsgSize is the maximum message size.
	maxMsgSize int
}

// NewGordian constructs a Gordian object.
func NewGordian(h Handler) *Gordian {
	g := &Gordian{
		fromClient: make(chan Message),
		clientCtrl: make(chan clientInfo),
		clients:    make(map[ClientId]clientInfo),
		handler:    h,
		maxMsgSize: DefaultMaxMsgSize,
	}
	return g
}

// SetMaxMsgSize sets the maximum size of messages.
// NOTE: This may only be called before Run.
func (g *Gordian) SetMaxMsgSize(size int) {
	g.maxMsgSize = size
}

// Run initiates the goroutine that manages client connections and message
// distribution.
func (g *Gordian) Run() {
	go g.manageClients()
}

// Send passes a message to the specified client.
func (g *Gordian) Send(id ClientId, msg Message) {
	if ci, ok := g.clients[id]; ok {
		ci.toClient <- msg
	}
}

// WSHandler returns a function to be called by http.ListenAndServe to handle
// a new websocket connection.
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

// manageClients waits for connection or message events and updates the
// internal state or delivers the message, respectively.
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

// readFromWS reads a message from the client and passes it to clientManager.
func (g *Gordian) readFromWS(ws *websocket.Conn, ci clientInfo) {
	msg := make(MessageData, g.maxMsgSize)
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

// readFromWS waits for messages from clientManager and sends them to the
// client.
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
