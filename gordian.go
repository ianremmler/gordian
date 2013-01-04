package gordian

import (
	"code.google.com/p/go.net/websocket"

	"fmt"
	"io"
)

const (
	CONNECT = iota
	REGISTER
	CLOSE
)

type ClientId interface{}
type MessageData interface{}

type Message struct {
	From ClientId
	To   ClientId
	Data MessageData
}

type Client struct {
	Id      ClientId
	Ctrl    int
	Conn    *websocket.Conn
	message chan *Message
}

type Gordian struct {
	Control chan *Client
	Message chan *Message
	manage  chan *Client
	clients map[ClientId]*Client
}

func New() *Gordian {
	g := &Gordian{
		Control: make(chan *Client),
		Message: make(chan *Message),
		manage:  make(chan *Client),
		clients: make(map[ClientId]*Client),
	}
	return g
}

func (g *Gordian) Run() {
	go g.manageClients()
}

func (g *Gordian) WSHandler() func(conn *websocket.Conn) {
	return func(conn *websocket.Conn) {
		g.Control <- &Client{Ctrl: CONNECT, Conn: conn}
		client := <-g.Control
		if client.Id == nil || client.Ctrl != REGISTER {
			return
		}
		client.message = make(chan *Message)
		g.manage <- client
		go g.writeToWS(client)
		g.readFromWS(client)
		client.Ctrl = CLOSE
		g.Control <- client
		g.manage <- client
	}
}

func (g *Gordian) manageClients() {
	for {
		select {
		case msg := <-g.Message:
			if client, ok := g.clients[msg.To]; ok {
				client.message <- msg
			}
		case client := <-g.manage:
			switch client.Ctrl {
			case REGISTER:
				g.clients[client.Id] = client
			case CLOSE:
				close(client.message)
				delete(g.clients, client.Id)
			}
		}
	}
}

func (g *Gordian) readFromWS(client *Client) {
	for {
		var data MessageData
		err := websocket.JSON.Receive(client.Conn, &data)
		switch err {
		case nil:
			g.Message <- &Message{client.Id, nil, data}
		case io.EOF:
			return
		default:
			fmt.Println(err)
		}
	}
}

func (g *Gordian) writeToWS(client *Client) {
	for {
		msg, ok := <-client.message
		if !ok {
			return
		}
		if err := websocket.JSON.Send(client.Conn, msg.Data); err != nil {
			fmt.Println(err)
		}
	}
}
