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

	"github.com/khoai-link-protocol/core"
)

func Start(addr string, opts ...Option) error {
	// 1. Doc options va load config
	options := &nodeOptions{
		configPath: "khoai-config.json",
	}
	for _, opt := range opts {
		opt(options)
	}

	cfg, err := LoadConfig(options.configPath)
	if err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}

	// 2. Khoi tao Clients
	p2pCli := client.NewP2pClient(15 * time.Second)
	mcpCli := client.NewMCPClient(cfg.MCPGatewayURL, 5*time.Second)

	// 3. Khoi tao Engine (truyen p2pClient vao de dispatch sang node khac)
	exec := engine.NewExecutor(cfg.NodeID, p2pCli)

	// 4. Khoi tao Server (truyen Engine vao handler)
	srv := server.NewP2pNode(addr, cfg.MCPGatewayURL, exec)

	// 5. Bat Server trong Goroutine rieng de khong block tien trinh
	serverErrChan := make(chan error, 1)
	go func() {
		if err := srv.StartNode(); err != nil {
			serverErrChan <- err
		}
	}()
	log.Println("start node", cfg.NodeID, "at addr", addr)

	// Cho 50-100ms de port TCP thuc su san sang truoc khi bao len MCP
	time.Sleep(100 * time.Millisecond)

	// 6. Dang ky Node len MCP Gateway
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cap := core.Capability{
		ID:       cfg.NodeID,
		Name:     cfg.NodeName, // hoac gan cfg.Name neu config co truong name
		IP:       addr,         // gan IP bang dia chi addr cua Node
		Tasks:    cfg.Tasks,
		Metadata: cfg.Metadata,
	}

	// Gui Capability sang MCP Gateway
	if _, err := mcpCli.MCPRegister(ctx, cap); err != nil {
		return fmt.Errorf("failed to register node with mcp gateway: %w", err)
	}

	// 7. Lang nghe tin hieu he dieu hanh de tat server an toan
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrChan:
		return fmt.Errorf("server crashed: %w", err)
	case <-shutdownChan:
		// Graceful shutdown
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer sCancel()
		return srv.Shutdown(shutdownCtx)
	}
}
