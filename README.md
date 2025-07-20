# Multi-Protocol Mock Server

A comprehensive mock server implementation in Go that supports HTTP, WebSocket, and gRPC protocols for testing API Gateway functionality.

## Features

### HTTP Server (Echo v4)
- **Health Check**: `GET /health` - Returns server health status
- **Echo Endpoints**: 
  - `GET /echo` - Echoes back request headers and query parameters
  - `POST /echo` - Echoes back JSON payload with headers
- **Delay Testing**: `GET /delay/:seconds` - Delayed response (0-30 seconds)
- **Status Testing**: `GET /status/:code` - Returns specific HTTP status codes (100-599)

### WebSocket Server (Echo v4 + Gorilla WebSocket)
- **Echo WebSocket**: `/ws/echo` - Echoes back messages
- **Broadcast WebSocket**: `/ws/broadcast` - Broadcasts to all connected clients
- **Chat Rooms**: `/ws/chat/:room` - Room-based chat functionality

### gRPC Server (Custom Implementation)
- **Unary RPC**: Simple request/response
- **Server Streaming**: Stream multiple responses
- **Client Streaming**: Accept stream of requests  
- **Bidirectional Streaming**: Full-duplex communication

## Quick Start

### Running with Go
```bash
# Run HTTP/WebSocket server only
go run cmd/server/main-http-ws.go

# Run full server with gRPC (requires gRPC fixes)
go run cmd/server/main.go
```

### Running with Docker
```bash
# Build and run
docker build -t mockserver .
docker run -p 8080:8080 -p 50051:50051 mockserver

# Or use docker-compose
docker-compose up
```

## API Testing Examples

### HTTP Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
# Response: {"status":"healthy","timestamp":1752996691}
```

#### Echo GET
```bash
curl http://localhost:8080/echo?param1=value1
# Response: {"method":"GET","path":"/echo","query":{"param1":["value1"]},"headers":{...},"timestamp":...}
```

#### Echo POST
```bash
curl -X POST http://localhost:8080/echo \
  -H "Content-Type: application/json" \
  -d '{"message":"test","value":123}'
# Response: {"method":"POST","path":"/echo","headers":{...},"body":{"message":"test","value":123},"timestamp":...}
```

#### Delay Testing
```bash
curl http://localhost:8080/delay/3
# Waits 3 seconds, then: {"delay_seconds":3,"message":"Response after delay","timestamp":...}
```

#### Status Code Testing
```bash
curl http://localhost:8080/status/404
# Response: {"status_code":404,"message":"Not Found","timestamp":...}
```

### WebSocket Testing

#### Echo WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/echo');
ws.onopen = () => {
    ws.send(JSON.stringify({
        type: "test",
        data: "Hello WebSocket!",
        timestamp: Date.now()
    }));
};
ws.onmessage = (event) => {
    console.log('Received:', JSON.parse(event.data));
    // Response: {type: "echo", data: "Hello WebSocket!", timestamp: ...}
};
```

#### Broadcast WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/broadcast');
// Messages sent to this endpoint are broadcast to all connected clients
```

#### Chat Room WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/chat/room1');
// Join room "room1" for group chat functionality
```

### gRPC Testing

Use tools like `grpcurl` or custom gRPC clients to test:

```bash
# Unary call
grpcurl -plaintext -d '{"message":"test","value":123}' localhost:50051 mock.MockService/Echo

# Server streaming
grpcurl -plaintext -d '{"id":"test","data":"stream test"}' localhost:50051 mock.MockService/ServerStream
```

## Docker Configuration

### Ports
- **8080**: HTTP/WebSocket server
- **50051**: gRPC server
- **4770**: gripmock gRPC (optional)
- **4771**: gripmock admin HTTP (optional)

### Environment Variables
- `LOG_LEVEL`: Set logging level (default: info)

## Development

### Project Structure
```
cmd/server/          # Main application entry points
internal/
├── http/           # HTTP handlers and server
├── websocket/      # WebSocket handlers
└── grpc/           # gRPC service implementation
proto/              # Protocol buffer definitions
test/               # Test utilities and examples
docker/             # Docker configurations
```

### Building
```bash
go mod tidy
go build -o mockserver cmd/server/main.go
```

### Testing
```bash
# Test WebSocket client
go run test/ws_client.go /ws/echo "Test message"
```

## Use Cases

This mock server is perfect for:

1. **API Gateway Testing**: Test routing, load balancing, and protocol translation
2. **Integration Testing**: Mock upstream services during development
3. **Performance Testing**: Generate consistent responses for load testing
4. **Protocol Testing**: Verify multi-protocol support in your infrastructure
5. **Development**: Local development environment for microservices

## Configuration

The server can be configured through:
- Environment variables
- Command line flags (can be added)
- Configuration files (can be extended)

## Health Checks

All services include health check endpoints:
- HTTP: `GET /health`
- Docker health checks included in docker-compose.yml

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## Status

✅ HTTP Server - Fully functional
✅ WebSocket Server - Fully functional  
🔧 gRPC Server - Implementation complete, needs protobuf compatibility fixes

The server successfully demonstrates multi-protocol capabilities and is ready for API Gateway testing scenarios.