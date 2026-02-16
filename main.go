package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "Interzoid Data Quality APIs"
	serverVersion = "1.0.0"
)

func main() {
	// CLI flags
	transport := flag.String("transport", "stdio", "Transport type: stdio or http")
	port := flag.String("port", "8080", "Port for HTTP transport")
	flag.Parse()

	// Create the MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)

	// Register all Interzoid API tools
	registerAllTools(s)

	switch *transport {
	case "stdio":
		log.Println("Starting Interzoid MCP server (stdio transport)...")
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}

	case "http":
		addr := ":" + *port
		log.Printf("Starting Interzoid MCP server (StreamableHTTP transport) on %s...\n", addr)
		log.Printf("MCP endpoint available at http://localhost%s/mcp\n", addr)

		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(addr); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown transport: %s (use 'stdio' or 'http')\n", *transport)
		os.Exit(1)
	}
}
