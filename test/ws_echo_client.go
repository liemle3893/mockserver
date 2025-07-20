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
	wsURL := os.Getenv("WS_ADDR")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8080"
	}

	u := url.URL{Scheme: "ws", Host: wsURL, Path: "/ws/echo"}
	log.Printf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// Send a test message
	msg := Message{
		Type:      "test",
		Data:      "Hello Echo WebSocket!",
		Timestamp: time.Now().Unix(),
	}

	log.Printf("Sending: %+v", msg)
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

	fmt.Printf("✅ Echo WebSocket Test PASSED\n")
	fmt.Printf("Sent: %+v\n", msg)
	fmt.Printf("Received: %+v\n", response)
}
