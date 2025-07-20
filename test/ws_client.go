package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	Room      string      `json:"room,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ws_client.go <endpoint> [message]")
	}

	endpoint := os.Args[1]
	message := "Hello WebSocket!"
	if len(os.Args) > 2 {
		message = os.Args[2]
	}

	wsURL := os.Getenv("WS_ADDR")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8080"
	}

	u := url.URL{Scheme: "ws", Host: wsURL, Path: endpoint}
	log.Printf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// Send a test message
	msg := Message{
		Type:      "test",
		Data:      message,
		Timestamp: time.Now().Unix(),
	}

	err = c.WriteJSON(msg)
	if err != nil {
		log.Println("write:", err)
		return
	}

	// Read response
	var response Message
	err = c.ReadJSON(&response)
	if err != nil {
		log.Println("read:", err)
		return
	}

	fmt.Printf("Received: %+v\n", response)
}
