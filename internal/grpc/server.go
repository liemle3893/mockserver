package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	pb "mockserver/proto"
)

type MockServer struct {
	pb.UnimplementedMockServiceServer
}

func NewMockServer() *MockServer {
	return &MockServer{}
}

// Echo implements unary RPC
func (s *MockServer) Echo(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	log.Printf("gRPC Echo: Received message: %s, value: %d", req.Message, req.Value)
	
	response := &pb.SimpleResponse{
		Message:   fmt.Sprintf("Echo: %s (value: %d)", req.Message, req.Value),
		Timestamp: time.Now().Unix(),
	}
	
	log.Printf("gRPC Echo: Sending response: %s", response.Message)
	return response, nil
}

// ServerStream implements server streaming RPC
func (s *MockServer) ServerStream(req *pb.StreamRequest, stream pb.MockService_ServerStreamServer) error {
	log.Printf("gRPC ServerStream: Starting stream for ID: %s, data: %s", req.Id, req.Data)
	
	// Send 5 responses with incremental sequence numbers
	for i := 0; i < 5; i++ {
		if err := stream.Context().Err(); err != nil {
			log.Printf("gRPC ServerStream: Context error: %v", err)
			return err
		}
		
		response := &pb.StreamResponse{
			Id:        req.Id,
			Data:      fmt.Sprintf("%s - response %d", req.Data, i+1),
			Timestamp: time.Now().Unix(),
			Sequence:  int32(i + 1),
		}
		
		if err := stream.Send(response); err != nil {
			log.Printf("gRPC ServerStream: Send error: %v", err)
			return err
		}
		
		log.Printf("gRPC ServerStream: Sent response %d: %s", i+1, response.Data)
		
		// Small delay between responses
		time.Sleep(100 * time.Millisecond)
	}
	
	log.Printf("gRPC ServerStream: Completed stream for ID: %s", req.Id)
	return nil
}

// ClientStream implements client streaming RPC
func (s *MockServer) ClientStream(stream pb.MockService_ClientStreamServer) error {
	log.Printf("gRPC ClientStream: Starting client stream")
	
	var messages []string
	var totalValue int32
	count := 0
	
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// End of stream, send response
			response := &pb.SimpleResponse{
				Message:   fmt.Sprintf("Received %d messages: %v (total value: %d)", count, messages, totalValue),
				Timestamp: time.Now().Unix(),
			}
			
			log.Printf("gRPC ClientStream: Sending final response: %s", response.Message)
			return stream.SendAndClose(response)
		}
		if err != nil {
			log.Printf("gRPC ClientStream: Receive error: %v", err)
			return err
		}
		
		messages = append(messages, req.Data)
		count++
		
		log.Printf("gRPC ClientStream: Received message %d: ID=%s, data=%s", count, req.Id, req.Data)
	}
}

// BidiStream implements bidirectional streaming RPC
func (s *MockServer) BidiStream(stream pb.MockService_BidiStreamServer) error {
	log.Printf("gRPC BidiStream: Starting bidirectional stream")
	
	var wg sync.WaitGroup
	errChan := make(chan error, 2)
	
	// Goroutine to receive messages from client
	wg.Add(1)
	go func() {
		defer wg.Done()
		sequence := int32(0)
		
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				log.Printf("gRPC BidiStream: Client closed stream")
				return
			}
			if err != nil {
				log.Printf("gRPC BidiStream: Receive error: %v", err)
				errChan <- err
				return
			}
			
			sequence++
			log.Printf("gRPC BidiStream: Received message %d: ID=%s, data=%s", sequence, req.Id, req.Data)
			
			// Echo back the message with modifications
			response := &pb.StreamResponse{
				Id:        req.Id,
				Data:      fmt.Sprintf("Echo: %s (processed)", req.Data),
				Timestamp: time.Now().Unix(),
				Sequence:  sequence,
			}
			
			if err := stream.Send(response); err != nil {
				log.Printf("gRPC BidiStream: Send error: %v", err)
				errChan <- err
				return
			}
			
			log.Printf("gRPC BidiStream: Sent response %d: %s", sequence, response.Data)
		}
	}()
	
	// Wait for receiving goroutine to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()
	
	// Wait for any errors or completion
	err := <-errChan
	if err != nil {
		log.Printf("gRPC BidiStream: Stream error: %v", err)
		return err
	}
	
	log.Printf("gRPC BidiStream: Stream completed")
	return nil
}