package main

import (
	"log"
	"os"

	"github.com/AmazingCYJ/AgentRAG/internal/platform/app"
)

func main() {
	configPath := os.Getenv("AGENTRAG_CONFIG")
	if configPath == "" {
		configPath = "configs/config.example.yaml"
	}

	instance, err := app.NewApp(configPath)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	log.Printf("api 服务启动中，配置文件: %s", configPath)
	instance.HTTPServer.Run()
}
