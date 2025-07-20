package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	Room      string      `json:"room,omitempty"`
}

func createClient(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	wsURL := os.Getenv("WS_ADDR")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8080"
	}

	u := url.URL{Scheme: "ws", Host: wsURL, Path: "/ws/broadcast"}
	log.Printf("Client %d: Connecting to %s", id, u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Client %d: dial error: %v", id, err)
		return
	}
	defer c.Close()

	// Start reading messages
	go func() {
		for {
			var response Message
			err := c.ReadJSON(&response)
			if err != nil {
				log.Printf("Client %d: read error: %v", id, err)
				return
			}
			fmt.Printf("🔔 Client %d received broadcast: %+v\n", id, response)
		}
	}()

	// Wait a bit to ensure all clients are connected
	time.Sleep(2 * time.Second)

	// Send a message from this client
	if id == 1 {
		msg := Message{
			Type:      "test",
			Data:      fmt.Sprintf("Broadcast message from client %d", id),
			Timestamp: time.Now().Unix(),
		}

		log.Printf("Client %d: Sending broadcast: %+v", id, msg)
		err = c.WriteJSON(msg)
		if err != nil {
			log.Printf("Client %d: write error: %v", id, err)
			return
		}
	}

	// Keep connection alive for a while
	time.Sleep(5 * time.Second)
}

func main() {
	var wg sync.WaitGroup

	// Create 3 clients
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go createClient(i, &wg)
	}

	wg.Wait()
	fmt.Println("✅ Broadcast WebSocket Test COMPLETED")
}
