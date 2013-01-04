package chat

import (
	"github.com/ianremmler/gordian"

	"strings"
)

type Chat struct {
	clients map[gordian.ClientId]struct{}
	*gordian.Gordian
}

func New() *Chat {
	return &Chat{
		clients: make(map[gordian.ClientId]struct{}),
		Gordian: gordian.New(),
	}
}

func (c *Chat) Run() {
	go c.run()
	c.Gordian.Run()
}

func (c *Chat) run() {
	for {
		select {
		case client := <-c.Control:
			switch client.Ctrl {
			case gordian.CONNECT:
				client.Ctrl = gordian.REGISTER
				if !c.connect(client) {
					client.Ctrl = gordian.CLOSE
				}
				c.Control <- client
			case gordian.CLOSE:
				c.close(client)
			}
		case m := <-c.Message:
			c.handleMessage(m)
		}
	}
}

func (c *Chat) connect(client *gordian.Client) bool {
	path := client.Conn.Request().URL.Path
	client.Id = path[strings.LastIndex(path, "/")+1:]
	if client.Id == "" {
		return false
	}
	c.clients[client.Id] = struct{}{}
	return true
}

func (c *Chat) close(client *gordian.Client) {
	delete(c.clients, client.Id)
}

func (c *Chat) send(msg *gordian.Message) {
	c.Message <- msg
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
