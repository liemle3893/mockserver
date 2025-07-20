package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpServer "mockserver/internal/http"
	wsHandlers "mockserver/internal/websocket"
)

const (
	HTTPPort = 8080
)

func main() {
	// Create HTTP/WebSocket server
	server := httpServer.NewServer(HTTPPort)
	
	// Add WebSocket handlers
	wsHandler := wsHandlers.NewWebSocketHandlers()
	server.SetupRoutes()
	
	// Add WebSocket routes
	e := server.GetEcho()
	e.GET("/ws/echo", wsHandler.Echo)
	e.GET("/ws/broadcast", wsHandler.Broadcast)
	e.GET("/ws/chat/:room", wsHandler.Chat)

	// Start HTTP/WebSocket server in a goroutine
	go func() {
		log.Printf("Starting HTTP/WebSocket server on port %d", HTTPPort)
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}