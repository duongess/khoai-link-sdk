package server

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

type P2pNode struct {
	Addr          string
	MCPGatewayURL string
	Peers         map[string]string
	mu            sync.RWMutex
	httpServer    *http.Server
}

func NewP2pNode(addr, mcpGatewayURL string) *P2pNode {
	return &P2pNode{
		Addr:          addr,
		MCPGatewayURL: mcpGatewayURL,
		Peers:         make(map[string]string),
	}
}

// Request payload mau cho endpoint execute
type ExecuteTaskRequest struct {
	TaskName string         `json:"task_name"`
	Inputs   map[string]any `json:"inputs"`
}

func (s *P2pNode) setupRoutes() *Router {
	rt := NewRouter()

	// 1. Health check endpoint (GET)
	RegisterGET(rt, "/", func(c *Context[EmptyBody]) (any, int, error) {
		return "The node is ready", http.StatusOK, nil
	})

	// 2. Peer discovery / sync endpoint (GET)
	RegisterGET(rt, "/api/v1/peers", func(c *Context[EmptyBody]) (any, int, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.Peers, http.StatusOK, nil
	})

	// 3. Task execution endpoint (POST)
	RegisterPOST(rt, "/api/v1/execute", func(c *Context[ExecuteTaskRequest]) (any, int, error) {
		if c.Body.TaskName == "" {
			return nil, http.StatusBadRequest, errors.New("task_name is required")
		}

		// Goi logic thuc thi task o day
		result := map[string]any{
			"task":   c.Body.TaskName,
			"status": "completed",
		}
		return result, http.StatusOK, nil
	})

	return rt
}

func (s *P2pNode) StartNode() error {
	router := s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         s.Addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s.httpServer.ListenAndServe()
}

func (s *P2pNode) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
