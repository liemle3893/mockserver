package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandlers struct {
	clients map[*websocket.Conn]bool
	rooms   map[string]map[*websocket.Conn]bool
	mutex   sync.RWMutex
}

func NewWebSocketHandlers() *WebSocketHandlers {
	return &WebSocketHandlers{
		clients: make(map[*websocket.Conn]bool),
		rooms:   make(map[string]map[*websocket.Conn]bool),
	}
}

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	Room      string      `json:"room,omitempty"`
}

type ErrorMessage struct {
	Type      string `json:"type"`
	Error     string `json:"error"`
	Details   string `json:"details,omitempty"`
	RawData   string `json:"raw_data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Helper function to safely read WebSocket JSON messages
func safeReadJSON(ws *websocket.Conn) (*Message, error) {
	// First, read the raw message
	messageType, data, err := ws.ReadMessage()
	if err != nil {
		return nil, err
	}

	// Only process text messages (JSON)
	if messageType != websocket.TextMessage {
		return &Message{
			Type:      "error",
			Data:      "Binary messages not supported",
			Timestamp: time.Now().Unix(),
		}, nil
	}

	// Try to parse as JSON
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		// Return an error message instead of failing
		return &Message{
			Type: "json_error",
			Data: map[string]interface{}{
				"error":    "Invalid JSON format",
				"details":  err.Error(),
				"raw_data": string(data),
			},
			Timestamp: time.Now().Unix(),
		}, nil
	}

	// Set timestamp if not provided
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	return &msg, nil
}

// Helper function to safely write JSON to WebSocket
func safeWriteJSON(ws *websocket.Conn, data interface{}) error {
	// Set a write deadline to prevent hanging
	ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return ws.WriteJSON(data)
}

// Echo WebSocket - echoes back messages with error handling
func (h *WebSocketHandlers) Echo(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}
	defer ws.Close()

	log.Printf("WebSocket Echo: New connection established")

	// Send welcome message
	welcome := Message{
		Type:      "welcome",
		Data:      "Connected to Echo WebSocket. Send any JSON message to echo it back.",
		Timestamp: time.Now().Unix(),
	}
	if err := safeWriteJSON(ws, welcome); err != nil {
		log.Printf("WebSocket Echo: Failed to send welcome message: %v", err)
		return nil
	}

	for {
		msg, err := safeReadJSON(ws)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket Echo read error: %v", err)
			}
			break
		}

		log.Printf("WebSocket Echo: Received message: %+v", msg)

		// If it's a JSON error, send the error back as is
		if msg.Type == "json_error" {
			if err := safeWriteJSON(ws, msg); err != nil {
				log.Printf("WebSocket Echo write error: %v", err)
				break
			}
			log.Printf("WebSocket Echo: Sent JSON error response")
			continue
		}

		// Normal echo response
		response := Message{
			Type:      "echo",
			Data:      msg.Data,
			Timestamp: time.Now().Unix(),
		}

		if err := safeWriteJSON(ws, response); err != nil {
			log.Printf("WebSocket Echo write error: %v", err)
			break
		}

		log.Printf("WebSocket Echo: Sent response: %+v", response)
	}

	log.Printf("WebSocket Echo: Connection closed")
	return nil
}

// Broadcast WebSocket - broadcasts to all connected clients with error handling
func (h *WebSocketHandlers) Broadcast(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}
	defer h.removeClient(ws)

	log.Printf("WebSocket Broadcast: New connection established")
	h.addClient(ws)

	// Send welcome message
	welcome := Message{
		Type:      "welcome",
		Data:      "Connected to Broadcast WebSocket. Your messages will be sent to all connected clients.",
		Timestamp: time.Now().Unix(),
	}
	if err := safeWriteJSON(ws, welcome); err != nil {
		log.Printf("WebSocket Broadcast: Failed to send welcome message: %v", err)
	}

	for {
		msg, err := safeReadJSON(ws)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket Broadcast read error: %v", err)
			}
			break
		}

		log.Printf("WebSocket Broadcast: Received message: %+v", msg)

		// If it's a JSON error, only send back to the sender
		if msg.Type == "json_error" {
			if err := safeWriteJSON(ws, msg); err != nil {
				log.Printf("WebSocket Broadcast write error: %v", err)
				break
			}
			log.Printf("WebSocket Broadcast: Sent JSON error response to sender only")
			continue
		}

		// Normal broadcast
		broadcast := Message{
			Type:      "broadcast",
			Data:      msg.Data,
			Timestamp: time.Now().Unix(),
		}

		h.broadcastToAll(broadcast)
		log.Printf("WebSocket Broadcast: Message broadcasted to all clients")
	}

	log.Printf("WebSocket Broadcast: Connection closed")
	return nil
}

// Chat WebSocket - room-based chat with error handling
func (h *WebSocketHandlers) Chat(c echo.Context) error {
	room := c.Param("room")
	if room == "" {
		log.Printf("WebSocket Chat: Missing room parameter")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Room parameter is required",
		})
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}
	defer h.removeFromRoom(ws, room)

	log.Printf("WebSocket Chat: New connection to room '%s'", room)
	h.addToRoom(ws, room)

	// Send welcome message to the new user
	welcome := Message{
		Type:      "welcome",
		Data:      map[string]string{"message": "Connected to room " + room + ". Send JSON messages to chat."},
		Timestamp: time.Now().Unix(),
		Room:      room,
	}
	if err := safeWriteJSON(ws, welcome); err != nil {
		log.Printf("WebSocket Chat: Failed to send welcome message: %v", err)
	}

	// Send join message to room
	joinMsg := Message{
		Type:      "join",
		Data:      map[string]string{"message": "User joined room " + room},
		Timestamp: time.Now().Unix(),
		Room:      room,
	}
	h.broadcastToRoom(room, joinMsg)
	log.Printf("WebSocket Chat: User joined room '%s'", room)

	for {
		msg, err := safeReadJSON(ws)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket Chat read error: %v", err)
			}
			break
		}

		log.Printf("WebSocket Chat: Received message in room '%s': %+v", room, msg)

		// If it's a JSON error, only send back to the sender
		if msg.Type == "json_error" {
			msg.Room = room // Add room info to error
			if err := safeWriteJSON(ws, msg); err != nil {
				log.Printf("WebSocket Chat write error: %v", err)
				break
			}
			log.Printf("WebSocket Chat: Sent JSON error response to sender in room '%s'", room)
			continue
		}

		// Normal chat message
		chatMsg := Message{
			Type:      "chat",
			Data:      msg.Data,
			Timestamp: time.Now().Unix(),
			Room:      room,
		}

		h.broadcastToRoom(room, chatMsg)
		log.Printf("WebSocket Chat: Message broadcasted to room '%s'", room)
	}

	// Send leave message to room
	leaveMsg := Message{
		Type:      "leave",
		Data:      map[string]string{"message": "User left room " + room},
		Timestamp: time.Now().Unix(),
		Room:      room,
	}
	h.broadcastToRoom(room, leaveMsg)
	log.Printf("WebSocket Chat: Connection to room '%s' closed", room)
	return nil
}

func (h *WebSocketHandlers) addClient(conn *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.clients[conn] = true
	log.Printf("WebSocket: Client added. Total clients: %d", len(h.clients))
}

func (h *WebSocketHandlers) removeClient(conn *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if _, exists := h.clients[conn]; exists {
		delete(h.clients, conn)
		conn.Close()
		log.Printf("WebSocket: Client removed. Total clients: %d", len(h.clients))
	}
}

func (h *WebSocketHandlers) addToRoom(conn *websocket.Conn, room string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*websocket.Conn]bool)
	}
	h.rooms[room][conn] = true
	log.Printf("WebSocket: Client added to room '%s'. Room size: %d", room, len(h.rooms[room]))
}

func (h *WebSocketHandlers) removeFromRoom(conn *websocket.Conn, room string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.rooms[room] != nil {
		if _, exists := h.rooms[room][conn]; exists {
			delete(h.rooms[room], conn)
			if len(h.rooms[room]) == 0 {
				delete(h.rooms, room)
				log.Printf("WebSocket: Room '%s' deleted (empty)", room)
			} else {
				log.Printf("WebSocket: Client removed from room '%s'. Room size: %d", room, len(h.rooms[room]))
			}
		}
	}
	conn.Close()
}

func (h *WebSocketHandlers) broadcastToAll(msg Message) {
	h.mutex.RLock()
	clientCount := len(h.clients)
	h.mutex.RUnlock()

	if clientCount == 0 {
		log.Printf("WebSocket Broadcast: No clients to broadcast to")
		return
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	successCount := 0
	for client := range h.clients {
		if err := safeWriteJSON(client, msg); err != nil {
			log.Printf("Broadcast error to client: %v", err)
			// Note: We can't modify the map here due to RLock, 
			// cleanup will happen when the client's read loop exits
		} else {
			successCount++
		}
	}
	log.Printf("WebSocket Broadcast: Message sent to %d/%d clients", successCount, clientCount)
}

func (h *WebSocketHandlers) broadcastToRoom(room string, msg Message) {
	h.mutex.RLock()
	roomClients := h.rooms[room]
	if roomClients == nil {
		h.mutex.RUnlock()
		log.Printf("WebSocket Room Broadcast: Room '%s' not found", room)
		return
	}
	
	clientCount := len(roomClients)
	h.mutex.RUnlock()

	if clientCount == 0 {
		log.Printf("WebSocket Room Broadcast: No clients in room '%s'", room)
		return
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	successCount := 0
	for client := range h.rooms[room] {
		if err := safeWriteJSON(client, msg); err != nil {
			log.Printf("Room broadcast error to client in room '%s': %v", room, err)
			// Note: We can't modify the map here due to RLock,
			// cleanup will happen when the client's read loop exits
		} else {
			successCount++
		}
	}
	log.Printf("WebSocket Room Broadcast: Message sent to %d/%d clients in room '%s'", successCount, clientCount, room)
}