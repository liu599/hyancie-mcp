package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	hyancieMCP "github.com/liu599/hyancie"
	"github.com/liu599/hyancie/logging"
	"github.com/liu599/hyancie/tools"

	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/cors"
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
	// 初始化日志
	if err := logging.InitLogger(); err != nil {
		return fmt.Errorf("初始化日志失败: %v", err)
	}

	// 加载配置
	if err := hyancieMCP.LoadConfig("config.json"); err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}
	defer func() {
		// Give the logger time to flush
		time.Sleep(1 * time.Second)
	}()

	s, err := newServer()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	switch transport {
	case "stdio":
		srv := server.NewStdioServer(s)
		logging.Logger.Info("Stdio server start")
		return srv.Listen(context.Background(), os.Stdin, os.Stdout)
	case "sse":
		url := hyancieMCP.Config.SseBaseUrl
		c := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		})

		srv := server.NewSSEServer(s,
			server.WithBaseURL(url),
		)
		logging.Logger.Info("SSE server listening on", "address", addr)
		handler := c.Handler(srv)
		if err := http.ListenAndServe(addr, handler); err != nil {
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
