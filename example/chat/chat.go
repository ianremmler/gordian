package main

import (
	"github.com/ianremmler/gordian/example/chatter"
	"net/http"
	"os"
	"websocket"
)

func main() {
	c := chatter.NewChatter()
	c.Run()

	chatDir := os.Getenv("CHAT_DIR")
	if chatDir == "" {
		chatDir = "/tmp"
	}
	http.Handle("/chat/", websocket.Handler(c.WSHandler()))
	http.Handle("/", http.FileServer(http.Dir(chatDir)))
	if err := http.ListenAndServe(":12345", nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
