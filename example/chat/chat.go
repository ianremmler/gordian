package chat

import (
	"code.google.com/p/go.net/websocket"
	"github.com/ianremmler/gordian"

	"errors"
	"strings"

	"fmt"
)

type Chat struct {
	clients map[gordian.ClientId]struct{}
	*gordian.Gordian
}

func New() *Chat {
	c := &Chat{
		clients: make(map[gordian.ClientId]struct{}),
		Gordian: gordian.New(),
	}
	return c
}

func (c *Chat) Run() {
	go c.run()
	c.Gordian.Run()
}

func (c *Chat) run() {
	for {
		select {
		case ws := <-c.Connect:
			id, err := c.connect(ws)
			c.Manage <- &gordian.ClientInfo{id, err == nil}
		case ci := <-c.Manage:
			if !ci.IsAlive {
				c.disconnect(ci.Id)
			}
		case m := <-c.Messages:
			c.handleMessage(m)
		}
	}
}

func (c *Chat) connect(ws *websocket.Conn) (gordian.ClientId, error) {
	path := ws.Request().URL.Path
	id := path[strings.LastIndex(path, "/")+1:]
	if id == "" {
		return "", errors.New("Invalid ID")
	}
	c.clients[id] = struct{}{}
	return id, nil
}

func (c *Chat) disconnect(id gordian.ClientId) {
	delete(c.clients, id)
}

func (c *Chat) send(msg *gordian.Message) {
	fmt.Println("sending:", msg)
	c.Messages <- msg
}

func (c *Chat) handleMessage(msg *gordian.Message) {
	data := msg.Data.(map[string]interface{})
	if in, ok := data["data"].(string); ok {
		data["data"] = msg.From.(string) + ": " + in
		out := &gordian.Message{From: msg.From, Data: data}
		for id, _ := range c.clients {
			out.To = id
			c.send(out)
		}
	}
}
