package khoailinksdk

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"khoai-link-sdk/client"
	"khoai-link-sdk/engine"
	"khoai-link-sdk/server"
	"khoai-link-sdk/types"

	"github.com/khoai-link-protocol/core"
)

type SDKNode struct {
	addr       string
	configPath string
	handlers   map[string]types.TaskHandler
}

func New(addr string, opts ...Option) *SDKNode {
	options := &nodeOptions{
		configPath: "khoai-config.json",
	}
	for _, opt := range opts {
		opt(options)
	}

	return &SDKNode{
		addr:       addr,
		configPath: options.configPath,
		handlers:   make(map[string]types.TaskHandler),
	}
}

// Dang ky task truoc khi start
func (n *SDKNode) RegisterTaskHandler(name string, handler types.TaskHandler) {
	n.handlers[name] = handler
}

func (n *SDKNode) Start() error {
	// 1. Load config
	cfg, err := LoadConfig(n.configPath)
	if err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}

	// 2. Khoi tao Clients
	p2pCli := client.NewP2pClient(15 * time.Second)
	mcpCli := client.NewMCPClient(cfg.MCPGatewayURL, 5*time.Second)

	// 3. Khoi tao Engine va nap toan bo handlers da dang ky vao
	exec := engine.NewExecutor(cfg.NodeID, p2pCli)
	for name, h := range n.handlers {
		exec.RegisterTaskHandler(name, h)
	}

	// 4. Khoi tao Server
	srv := server.NewP2pNode(n.addr, cfg.MCPGatewayURL, exec)

	// 5. Bat Server
	serverErrChan := make(chan error, 1)
	go func() {
		if err := srv.StartNode(); err != nil {
			serverErrChan <- err
		}
	}()
	log.Println("start node", cfg.NodeID, "at addr", n.addr)

	time.Sleep(100 * time.Millisecond)

	// 6. Dang ky Capability voi MCP Gateway
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cap := core.Capability{
		ID:       cfg.NodeID,
		Name:     cfg.NodeName,
		IP:       n.addr,
		Tasks:    cfg.Tasks,
		Metadata: cfg.Metadata,
	}

	if _, err := mcpCli.MCPRegister(ctx, cap); err != nil {
		return fmt.Errorf("failed to register node with mcp gateway: %w", err)
	}

	// 7. Lang nghe tin hieu he dieu hanh
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrChan:
		return fmt.Errorf("server crashed: %w", err)
	case <-shutdownChan:
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer sCancel()
		return srv.Shutdown(shutdownCtx)
	}
}
