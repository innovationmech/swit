package main

import (
	"log"

	switauth "github.com/innovationmech/swit/internal/switauth"
)

func main() {
	server, err := switauth.NewServer()
	if err != nil {
		log.Fatalf("无法创建服务器: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("服务器运行失败: %v", err)
	}
}
