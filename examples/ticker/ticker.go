package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/ianremmler/gordian"

	"net/http"
	"go/build"
	"strconv"
	"time"
)

type Ticker struct {
	clients map[gordian.ClientId]struct{}
	ticker  <-chan time.Time
	curId   int
	*gordian.Gordian
}

func NewTicker() *Ticker {
	t := &Ticker{
		clients: make(map[gordian.ClientId]struct{}),
		ticker:  time.Tick(10 * time.Millisecond),
		Gordian: gordian.New(),
	}
	return t
}

func (t *Ticker) Run() {
	go t.run()
	t.Gordian.Run()
}

func (t *Ticker) run() {
	msg := &gordian.Message{}
	data := map[string]interface{}{"type": "message"}
	i := 0
	for {
		select {
		case client := <-t.Control:
			switch client.Ctrl {
			case gordian.CONNECT:
				t.curId++
				client.Id = t.curId
				client.Ctrl = gordian.REGISTER
				t.clients[client.Id] = struct{}{}
				t.Control <- client
			case gordian.CLOSE:
				delete(t.clients, client.Id)
			}
		case <-t.ticker:
			data["data"] = strconv.Itoa(i)
			msg.Data = data
			for id, _ := range t.clients {
				msg.To = id
				t.Message <- msg
			}
			i++
		}
	}
}

func main() {
	t := NewTicker()
	t.Run()

	htmlDir := build.Default.GOPATH + "/src/github.com/ianremmler/gordian/examples/ticker"
	http.Handle("/ticker/", websocket.Handler(t.WSHandler()))
	http.Handle("/", http.FileServer(http.Dir(htmlDir)))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
