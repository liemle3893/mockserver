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

func createChatClient(id int, room string, wg *sync.WaitGroup) {
	defer wg.Done()

	wsURL := os.Getenv("WS_ADDR")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8080"
	}

	u := url.URL{Scheme: "ws", Host: wsURL, Path: fmt.Sprintf("/ws/chat/%s", room)}
	log.Printf("Client %d: Connecting to room '%s' at %s", id, room, u.String())

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
			fmt.Printf("💬 Client %d in room '%s' received: %+v\n", id, room, response)
		}
	}()

	// Wait a bit to ensure connection is established
	time.Sleep(1 * time.Second)

	// Send messages from different clients at different times
	if id == 1 {
		time.Sleep(2 * time.Second)
		msg := Message{
			Type:      "test",
			Data:      fmt.Sprintf("Hello from client %d in room %s", id, room),
			Timestamp: time.Now().Unix(),
		}

		log.Printf("Client %d: Sending chat message: %+v", id, msg)
		err = c.WriteJSON(msg)
		if err != nil {
			log.Printf("Client %d: write error: %v", id, err)
			return
		}
	}

	if id == 2 {
		time.Sleep(4 * time.Second)
		msg := Message{
			Type:      "test",
			Data:      fmt.Sprintf("Response from client %d in room %s", id, room),
			Timestamp: time.Now().Unix(),
		}

		log.Printf("Client %d: Sending chat message: %+v", id, msg)
		err = c.WriteJSON(msg)
		if err != nil {
			log.Printf("Client %d: write error: %v", id, err)
			return
		}
	}

	// Keep connection alive
	time.Sleep(8 * time.Second)
}

func main() {
	var wg sync.WaitGroup

	// Create clients in different rooms
	rooms := []string{"room1", "room1", "room2"}

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go createChatClient(i, rooms[i-1], &wg)
	}

	wg.Wait()
	fmt.Println("✅ Chat WebSocket Test COMPLETED")
}
