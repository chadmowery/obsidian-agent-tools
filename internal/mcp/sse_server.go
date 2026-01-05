package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// SSEServer wraps an MCP server with HTTP/SSE transport
type SSEServer struct {
	mcpServer *server.MCPServer
	clients   map[string]*sseClient
	mu        sync.RWMutex
}

type sseClient struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan bool
	lastSeen time.Time
}

// NewSSEServer creates a new SSE server
func NewSSEServer(mcpServer *server.MCPServer) *SSEServer {
	return &SSEServer{
		mcpServer: mcpServer,
		clients:   make(map[string]*sseClient),
	}
}

// ServeHTTP handles HTTP requests
func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.URL.Path {
	case "/mcp":
		s.handleMCPRequest(w, r)
	case "/mcp/sse":
		s.handleSSE(w, r)
	case "/health":
		s.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleHealth returns server health status
func (s *SSEServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"clients": len(s.clients),
	})
}

// handleMCPRequest handles MCP tool calls via POST
func (s *SSEServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse MCP request
	var request map[string]interface{}
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Handle different MCP methods
	method, ok := request["method"].(string)
	if !ok {
		http.Error(w, "Missing method", http.StatusBadRequest)
		return
	}

	var response interface{}

	switch method {
	case "tools/list":
		response = map[string]interface{}{
			"tools": s.mcpServer.ListTools(),
		}
	case "resources/list":
		response = map[string]interface{}{
			"resources": []interface{}{},
		}
	default:
		http.Error(w, fmt.Sprintf("Method not supported via HTTP: %s. Use stdio transport for full MCP support.", method), http.StatusNotImplemented)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSSE handles Server-Sent Events connections
func (s *SSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create client
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &sseClient{
		id:       clientID,
		writer:   w,
		flusher:  flusher,
		done:     make(chan bool),
		lastSeen: time.Now(),
	}

	// Register client
	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"clientId\":\"%s\"}\n\n", clientID)
	flusher.Flush()

	log.Printf("SSE client connected: %s", clientID)

	// Keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send keepalive
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-client.done:
			// Client disconnected
			s.mu.Lock()
			delete(s.clients, clientID)
			s.mu.Unlock()
			log.Printf("SSE client disconnected: %s", clientID)
			return

		case <-r.Context().Done():
			// Request cancelled
			s.mu.Lock()
			delete(s.clients, clientID)
			s.mu.Unlock()
			log.Printf("SSE client cancelled: %s", clientID)
			return
		}
	}
}

// Broadcast sends a message to all connected SSE clients
func (s *SSEServer) Broadcast(message interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	for _, client := range s.clients {
		fmt.Fprintf(client.writer, "data: %s\n\n", data)
		client.flusher.Flush()
	}
}
