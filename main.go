package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/way11229/aws_cli_execution/services"
)

func main() {
	// 創建服務器
	server := services.NewAWSCommandServer()
	server.Start(":80")

	// 優雅關機處理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 停止服務器
	server.Stop()
}
