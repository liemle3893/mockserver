package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("Testing malformed JSON handling...")

	// Wait a moment for server to be ready
	time.Sleep(3 * time.Second)

	baseURL := os.Getenv("HTTP_ADDR")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	wsURL := os.Getenv("WS_ADDR")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:8080"
	}

	// Test HTTP malformed JSON
	testHTTPMalformedJSON(baseURL)

	// Test WebSocket malformed JSON
	testWebSocketMalformedJSON(wsURL)

	fmt.Println("\nAll malformed JSON tests completed!")
}

func testHTTPMalformedJSON(baseURL string) {
	fmt.Println("\n=== Testing HTTP Malformed JSON ===")

	testCases := []struct {
		name string
		body string
	}{
		{"Empty body", ""},
		{"Valid JSON", `{"message": "hello", "number": 42}`},
		{"Malformed JSON - missing quote", `{"message: "hello"}`},
		{"Malformed JSON - trailing comma", `{"message": "hello",}`},
		{"Malformed JSON - unclosed brace", `{"message": "hello"`},
		{"Malformed JSON - invalid value", `{"message": hello}`},
		{"Plain text", "This is not JSON at all"},
		{"Partial JSON", `{"incomplete"`},
	}

	for _, tc := range testCases {
		fmt.Printf("\nTesting: %s\n", tc.name)
		fmt.Printf("Sending: %s\n", tc.body)

		resp, err := http.Post(baseURL+"/echo", "application/json", bytes.NewBufferString(tc.body))
		if err != nil {
			fmt.Printf("❌ Request failed: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("❌ Failed to decode response: %v\n", err)
			continue
		}

		fmt.Printf("✅ Status: %d\n", resp.StatusCode)
		if jsonError, exists := result["json_parse_error"]; exists {
			fmt.Printf("✅ JSON error handled gracefully: %v\n", jsonError)
		} else if result["body"] != nil {
			fmt.Printf("✅ Valid JSON parsed successfully\n")
		} else {
			fmt.Printf("✅ Empty body handled correctly\n")
		}
	}
}

func testWebSocketMalformedJSON(wsURL string) {
	fmt.Println("\n=== Testing WebSocket Malformed JSON ===")

	testCases := []struct {
		name string
		data string
	}{
		{"Valid JSON", `{"type": "test", "data": "hello"}`},
		{"Malformed JSON - missing quote", `{"type": test", "data": "hello"}`},
		{"Malformed JSON - trailing comma", `{"type": "test", "data": "hello",}`},
		{"Malformed JSON - unclosed brace", `{"type": "test", "data": "hello"`},
		{"Plain text", "This is not JSON"},
		{"Partial JSON", `{"incomplete`},
	}

	// Test Echo WebSocket
	fmt.Println("\n--- Testing Echo WebSocket ---")
	testWebSocketEndpoint(wsURL+"/ws/echo", testCases)

	// Test Broadcast WebSocket
	fmt.Println("\n--- Testing Broadcast WebSocket ---")
	testWebSocketEndpoint(wsURL+"/ws/broadcast", testCases)

	// Test Chat WebSocket
	fmt.Println("\n--- Testing Chat WebSocket ---")
	testWebSocketEndpoint(wsURL+"/ws/chat/test-room", testCases)
}

func testWebSocketEndpoint(endpoint string, testCases []struct{ name, data string }) {
	u, _ := url.Parse(endpoint)

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("❌ Failed to connect to %s: %v\n", endpoint, err)
		return
	}
	defer conn.Close()

	// Read welcome message
	var welcome map[string]interface{}
	if err := conn.ReadJSON(&welcome); err == nil {
		fmt.Printf("📨 Welcome: %v\n", welcome["data"])
	}

	for _, tc := range testCases {
		fmt.Printf("\nTesting: %s\n", tc.name)
		fmt.Printf("Sending: %s\n", tc.data)

		// Send raw message
		if err := conn.WriteMessage(websocket.TextMessage, []byte(tc.data)); err != nil {
			fmt.Printf("❌ Failed to send message: %v\n", err)
			continue
		}

		// Read response
		var response map[string]interface{}
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := conn.ReadJSON(&response); err != nil {
			fmt.Printf("❌ Failed to read response: %v\n", err)
			continue
		}

		if response["type"] == "json_error" {
			fmt.Printf("✅ JSON error handled gracefully: %v\n", response["data"])
		} else {
			fmt.Printf("✅ Valid message processed: type=%v\n", response["type"])
		}
	}
}
