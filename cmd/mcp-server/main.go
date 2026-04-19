package main

import (
	"log"
	"net/http"
	"os"

	"github.com/AmazingCYJ/AgentRAG/internal/mcpserver"
)

func main() {
	port := os.Getenv("AGENTRAG_MCP_PORT")
	if port == "" {
		port = "9099"
	}
	addr := ":" + port
	log.Printf("mcp server 启动中，监听地址: %s", addr)
	if err := http.ListenAndServe(addr, mcpserver.NewHTTPHandler()); err != nil {
		log.Fatalf("mcp server 启动失败: %v", err)
	}
}
