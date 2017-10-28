package main

import (
	"github.com/ianremmler/gordian"

	"go/build"
	"net/http"
	"os"
	"strings"
)

type Chat struct {
	clients map[gordian.ClientId]struct{}
	*gordian.Gordian
}

func NewChat() *Chat {
	return &Chat{
		clients: map[gordian.ClientId]struct{}{},
		Gordian: gordian.New(0),
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
			c.clientCtrl(client)
		case msg := <-c.InBox:
			c.handleMessage(&msg)
		}
	}
}

func (c *Chat) clientCtrl(client *gordian.Client) {
	switch client.Ctrl {
	case gordian.Connect:
		c.connect(client)
	case gordian.Close:
		c.close(client)
	}
}

func (c *Chat) connect(client *gordian.Client) {
	path := client.Request.URL.Path
	client.Id = path[strings.LastIndex(path, "/")+1:]
	if client.Id == "" {
		client.Ctrl = gordian.Abort
		c.Control <- client
		return
	}
	client.Ctrl = gordian.Register
	c.Control <- client
	client = <-c.Control
	if client.Ctrl != gordian.Establish {
		return
	}
	c.clients[client.Id] = struct{}{}
	return
}

func (c *Chat) close(client *gordian.Client) {
	delete(c.clients, client.Id)
}

func (c *Chat) send(msg gordian.Message) {
	c.OutBox <- msg
}

func (c *Chat) handleMessage(msg *gordian.Message) {
	inText := ""
	err := msg.Unmarshal(&inText)
	if err != nil {
		return
	}
	outText := msg.From.(string) + ": " + inText
	out := gordian.Message{Type: "message", From: msg.From, Data: outText}
	for id, _ := range c.clients {
		out.To = id
		c.send(out)
	}
}

func main() {
	c := NewChat()
	c.Run()

	htmlDir := build.Default.GOPATH + "/src/github.com/ianremmler/gordian/examples/chat"
	http.Handle("/chat/", c)
	http.Handle("/", http.FileServer(http.Dir(htmlDir)))
	port := ":8000"
	if len(os.Args) > 1 {
		port = ":" + os.Args[1]
	}
	if err := http.ListenAndServe(port, nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
