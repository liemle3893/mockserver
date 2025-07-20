package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcServer "mockserver/internal/grpc"
	httpHandlers "mockserver/internal/http"
	wsHandlers "mockserver/internal/websocket"
	pb "mockserver/proto"
)

func main() {
	log.Println("Starting Multi-Protocol Mock Server...")

	// Create handlers
	httpHandler := httpHandlers.NewHTTPHandlers()
	wsHandler := wsHandlers.NewWebSocketHandlers()
	grpcHandler := grpcServer.NewMockServer()

	// Setup Echo server for HTTP and WebSocket
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	// HTTP routes
	e.GET("/health", httpHandler.Health)
	e.GET("/echo", httpHandler.EchoGet)
	e.POST("/echo", httpHandler.EchoPost)
	e.GET("/delay/:seconds", httpHandler.Delay)
	e.GET("/status/:code", httpHandler.Status)

	// WebSocket routes
	e.GET("/ws/echo", wsHandler.Echo)
	e.GET("/ws/broadcast", wsHandler.Broadcast)
	e.GET("/ws/chat/:room", wsHandler.Chat)

	// Setup gRPC server
	grpcSrv := grpc.NewServer()
	pb.RegisterMockServiceServer(grpcSrv, grpcHandler)
	reflection.Register(grpcSrv) // Enable gRPC reflection

	// Create listeners
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50051"
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}

	var wg sync.WaitGroup

	// Start gRPC server in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("gRPC server starting on %s", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start HTTP/WebSocket server in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("HTTP/WebSocket server starting on %s", httpAddr)
		if err := e.Start(httpAddr); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Log server information
	log.Println("═══════════════════════════════════════")
	log.Println("🚀 Multi-Protocol Mock Server Running")
	log.Println("═══════════════════════════════════════")
	log.Printf("📡 HTTP/WebSocket: http://localhost%s", httpAddr)
	log.Printf("🔗 gRPC:           localhost%s", grpcAddr)
	log.Println("")
	log.Println("HTTP Endpoints:")
	log.Printf("  GET  %s/health", httpAddr)
	log.Printf("  GET  %s/echo", httpAddr)
	log.Printf("  POST %s/echo", httpAddr)
	log.Printf("  GET  %s/delay/:seconds", httpAddr)
	log.Printf("  GET  %s/status/:code", httpAddr)
	log.Println("")
	log.Println("WebSocket Endpoints:")
	log.Printf("  WS   ws://localhost%s/ws/echo", httpAddr)
	log.Printf("  WS   ws://localhost%s/ws/broadcast", httpAddr)
	log.Printf("  WS   ws://localhost%s/ws/chat/:room", httpAddr)
	log.Println("")
	log.Println("gRPC Service:")
	log.Printf("  GRPC localhost%s (MockService)", grpcAddr)
	log.Println("  - Echo (unary)")
	log.Println("  - ServerStream (server streaming)")
	log.Println("  - ClientStream (client streaming)")
	log.Println("  - BidiStream (bidirectional streaming)")
	log.Println("═══════════════════════════════════════")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := e.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Shutdown gRPC server
	grpcSrv.GracefulStop()

	log.Println("Servers stopped")
}
