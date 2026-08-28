package server

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"khoai-link-sdk/engine"

	"github.com/khoai-link-protocol/core"
)

type P2pNode struct {
	Addr          string
	MCPGatewayURL string
	Peers         map[string]string
	Engine        *engine.Executor
	mu            sync.RWMutex
	httpServer    *http.Server
}

func NewP2pNode(addr, mcpGatewayURL string, exec *engine.Executor) *P2pNode {
	return &P2pNode{
		Addr:          addr,
		MCPGatewayURL: mcpGatewayURL,
		Engine:        exec,
		Peers:         make(map[string]string),
	}
}

type ExecuteRequestPayload struct {
	ExecutionPlan  *core.ExecutionPlan       `json:"execution_plan"`
	CurrentStepID  string                    `json:"current_step_id,omitempty"`
	RuntimeOutputs map[string]map[string]any `json:"runtime_outputs,omitempty"`
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
	RegisterPOST(rt, "/api/v1/execute", func(c *Context[ExecuteRequestPayload]) (any, int, error) {
		if c.Body.ExecutionPlan == nil {
			return nil, http.StatusBadRequest, errors.New("execution_plan is required")
		}

		// Khong gan cung Nodes[0] o day nua.
		// Truyen truc tiep CurrentStepID (co the rong neu la Gateway goi khoi tao) vao Engine
		result, err := s.Engine.ExecuteAndDispatch(
			c.Req.Context(),
			c.RequestID,
			c.Body.ExecutionPlan,
			c.Body.CurrentStepID,
			c.Body.RuntimeOutputs,
		)
		if err != nil {
			return nil, http.StatusInternalServerError, err
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
