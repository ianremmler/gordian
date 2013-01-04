package sim

import (
	"github.com/ianremmler/gordian"

	"strconv"
	"time"
)

type Sim struct {
	clients map[gordian.ClientId]struct{}
	ticker  <-chan time.Time
	curId   int
	*gordian.Gordian
}

func New() *Sim {
	s := &Sim{
		clients: make(map[gordian.ClientId]struct{}),
		ticker:  time.Tick(10 * time.Millisecond),
		Gordian: gordian.New(),
	}
	return s
}

func (s *Sim) Run() {
	go s.run()
	s.Gordian.Run()
}

func (s *Sim) run() {
	msg := &gordian.Message{}
	data := map[string]interface{}{"type": "message"}
	i := 0
	for {
		select {
		case client := <-s.Control:
			switch client.Ctrl {
			case gordian.CONNECT:
				s.curId++
				client.Id = s.curId
				client.Ctrl = gordian.REGISTER
				s.clients[client.Id] = struct{}{}
				s.Control <- client
			case gordian.CLOSE:
				delete(s.clients, client.Id)
			}
		case <-s.ticker:
			data["data"] = strconv.Itoa(i)
			msg.Data = data
			for id, _ := range s.clients {
				msg.To = id
				s.Message <- msg
			}
			i++
		}
	}
}
