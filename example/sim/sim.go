package sim

import (
	"github.com/ianremmler/gordian"

	"fmt"
	"strconv"
	"time"
)

type Sim struct {
	clients map[gordian.ClientId]struct{}
	ticker  <-chan time.Time
	*gordian.Gordian
}

func New() *Sim {
	s := &Sim{
		clients: make(map[gordian.ClientId]struct{}),
		ticker:  time.Tick(10 * 1000000),
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
		case <-s.Connect:
			id := len(s.clients) + 1
			s.clients[id] = struct{}{}
			s.Manage <- &gordian.ClientInfo{id, true}
		case ci := <-s.Manage:
			if !ci.IsAlive {
				delete(s.clients, ci.Id)
			}
		case <-s.ticker:
			data["data"] = strconv.Itoa(i)
			msg.Data = data
			for id, _ := range s.clients {
				msg.To = id
				s.Messages <- msg
			}
			i++
		}
	}
}
