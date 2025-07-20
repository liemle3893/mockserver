package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "mockserver/proto"
)

func main() {
	// Connect to gRPC server
	serverAddr := os.Getenv("GRPC_ADDR")
	if serverAddr == "" {
		serverAddr = "qa3lp1n3acc76:50051" // Use container internal address
	}

	fmt.Printf("Connecting to gRPC server at %s...\n", serverAddr)

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewMockServiceClient(conn)

	fmt.Println("✅ Connected to gRPC server!")
	fmt.Println("🧪 Testing gRPC endpoints...")
	fmt.Println("═══════════════════════════════════════")

	// Test Unary RPC
	testEcho(client)

	// Test Server Streaming RPC
	testServerStream(client)

	// Test Client Streaming RPC
	testClientStream(client)

	// Test Bidirectional Streaming RPC
	testBidiStream(client)

	fmt.Println("═══════════════════════════════════════")
	fmt.Println("✅ All gRPC tests completed!")
}

func testEcho(client pb.MockServiceClient) {
	fmt.Println("\n📞 Testing Unary RPC - Echo")
	fmt.Println("───────────────────────────────────────")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.SimpleRequest{
		Message: "Hello gRPC!",
		Value:   42,
	}

	fmt.Printf("Sending: message='%s', value=%d\n", req.Message, req.Value)

	resp, err := client.Echo(ctx, req)
	if err != nil {
		log.Printf("❌ Echo error: %v", err)
		return
	}

	fmt.Printf("✅ Received: message='%s', timestamp=%d\n", resp.Message, resp.Timestamp)
}

func testServerStream(client pb.MockServiceClient) {
	fmt.Println("\n📡 Testing Server Streaming RPC - ServerStream")
	fmt.Println("───────────────────────────────────────")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.StreamRequest{
		Id:   "stream-test-1",
		Data: "Server streaming data",
	}

	fmt.Printf("Sending: id='%s', data='%s'\n", req.Id, req.Data)

	stream, err := client.ServerStream(ctx, req)
	if err != nil {
		log.Printf("❌ ServerStream error: %v", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("✅ Server stream completed")
			break
		}
		if err != nil {
			log.Printf("❌ ServerStream receive error: %v", err)
			break
		}

		fmt.Printf("📨 Received: id='%s', data='%s', sequence=%d, timestamp=%d\n",
			resp.Id, resp.Data, resp.Sequence, resp.Timestamp)
	}
}

func testClientStream(client pb.MockServiceClient) {
	fmt.Println("\n📤 Testing Client Streaming RPC - ClientStream")
	fmt.Println("───────────────────────────────────────")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.ClientStream(ctx)
	if err != nil {
		log.Printf("❌ ClientStream error: %v", err)
		return
	}

	// Send multiple messages
	messages := []string{
		"First message",
		"Second message",
		"Third message",
		"Final message",
	}

	for i, msg := range messages {
		req := &pb.StreamRequest{
			Id:   fmt.Sprintf("client-stream-%d", i+1),
			Data: msg,
		}

		fmt.Printf("📤 Sending %d: id='%s', data='%s'\n", i+1, req.Id, req.Data)

		if err := stream.Send(req); err != nil {
			log.Printf("❌ ClientStream send error: %v", err)
			return
		}

		time.Sleep(200 * time.Millisecond) // Small delay between sends
	}

	// Close and receive response
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Printf("❌ ClientStream close error: %v", err)
		return
	}

	fmt.Printf("✅ Final response: message='%s', timestamp=%d\n", resp.Message, resp.Timestamp)
}

func testBidiStream(client pb.MockServiceClient) {
	fmt.Println("\n🔄 Testing Bidirectional Streaming RPC - BidiStream")
	fmt.Println("───────────────────────────────────────")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stream, err := client.BidiStream(ctx)
	if err != nil {
		log.Printf("❌ BidiStream error: %v", err)
		return
	}

	var wg sync.WaitGroup

	// Goroutine to send messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer stream.CloseSend()

		messages := []string{
			"Bidirectional message 1",
			"Bidirectional message 2",
			"Bidirectional message 3",
			"Final bidirectional message",
		}

		for i, msg := range messages {
			req := &pb.StreamRequest{
				Id:   fmt.Sprintf("bidi-%d", i+1),
				Data: msg,
			}

			fmt.Printf("📤 Sending: id='%s', data='%s'\n", req.Id, req.Data)

			if err := stream.Send(req); err != nil {
				log.Printf("❌ BidiStream send error: %v", err)
				return
			}

			time.Sleep(500 * time.Millisecond) // Delay between sends
		}

		fmt.Println("📤 Finished sending messages")
	}()

	// Goroutine to receive messages
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				fmt.Println("✅ Bidirectional stream completed")
				return
			}
			if err != nil {
				log.Printf("❌ BidiStream receive error: %v", err)
				return
			}

			fmt.Printf("📨 Received: id='%s', data='%s', sequence=%d, timestamp=%d\n",
				resp.Id, resp.Data, resp.Sequence, resp.Timestamp)
		}
	}()

	wg.Wait()
}
