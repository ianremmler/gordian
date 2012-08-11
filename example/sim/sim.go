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
			s.curId++
			s.clients[s.curId] = struct{}{}
			s.Control <- &gordian.ClientInfo{s.curId, gordian.CONNECT}
		case ci := <-s.Control:
			if ci.CtrlType == gordian.DISCONNECT {
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
