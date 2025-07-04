package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	hyancieMCP "github.com/liu599/hyancie"
	"github.com/liu599/hyancie/tools"

	"github.com/mark3labs/mcp-go/server"
)

func newServer() (*server.MCPServer, error) {
	s := server.NewMCPServer(
		hyancieMCP.Config.ServerName,
		hyancieMCP.Config.ServerVersion,
	)

	// Add generic tools from config.json
	if err := tools.AddGenericTools(s); err != nil {
		return nil, fmt.Errorf("failed to add generic tools: %w", err)
	}

	return s, nil
}

func run(transport, addr string) error {
	// 加载配置
	if err := hyancieMCP.LoadConfig(); err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	s, err := newServer()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	switch transport {
	case "stdio":
		srv := server.NewStdioServer(s)
		log.Printf("Stdio server start")
		return srv.Listen(context.Background(), os.Stdin, os.Stdout)
	case "sse":
		url := "http://" + addr
		srv := server.NewSSEServer(s,
			server.WithBaseURL(url),
		)
		log.Printf("SSE server listening on %s", addr)
		if err := srv.Start(addr); err != nil {
			return fmt.Errorf("Server error: %v", err)
		}
	default:
		return fmt.Errorf(
			"Invalid transport type: %s. Must be 'stdio' or 'sse'",
			transport,
		)
	}
	return nil
}

func main() {
	var transport string
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(
		&transport,
		"transport",
		"stdio",
		"Transport type (stdio or sse)",
	)
	// Set default SSE address from config, but allow override from command line.
	addr := flag.String("sse-address", hyancieMCP.Config.SseAddress, "The host and port to start the sse server on")
	flag.Parse()

	if err := run(transport, *addr); err != nil {
		panic(err)
	}
}
