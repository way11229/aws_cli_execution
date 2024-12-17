package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type AWSCommandExecutor struct {
	profile string
	region  string
}

type CommandRequest struct {
	Command string `json:"command" binding:"required"`
}

type CommandResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func NewAWSCommandExecutor(profile, region string) *AWSCommandExecutor {
	return &AWSCommandExecutor{
		profile: profile,
		region:  region,
	}
}

func (e *AWSCommandExecutor) ExecuteAWSCommand(command string) (string, error) {
	// 使用 shell 執行完整的 AWS CLI 命令
	cmd := exec.Command("bash", "-c", command)

	// 設置環境變量
	if e.profile != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_PROFILE=%s", e.profile))
	}
	if e.region != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_DEFAULT_REGION=%s", e.region))
	}

	// 執行命令並捕獲輸出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("執行 AWS CLI 命令出錯: %v, 輸出: %s", err, string(output))
	}

	return string(output), nil
}

type AWSCommandServer struct {
	router   *gin.Engine
	server   *http.Server
	wg       sync.WaitGroup
	executor *AWSCommandExecutor
}

func NewAWSCommandServer() *AWSCommandServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 創建 AWS 命令執行器
	executor := NewAWSCommandExecutor("default", "ap-northeast-1")

	server := &AWSCommandServer{
		router:   router,
		executor: executor,
	}

	// 設置路由
	router.POST("/execute-aws-command", server.handleAWSCommand)

	return server
}

func (s *AWSCommandServer) handleAWSCommand(c *gin.Context) {
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 執行 AWS 命令
	output, err := s.executor.ExecuteAWSCommand(req.Command)

	if err != nil {
		c.JSON(http.StatusInternalServerError, CommandResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, CommandResponse{
		Output: output,
	})
}

func (s *AWSCommandServer) Start(port string) {
	s.server = &http.Server{
		Addr:    port,
		Handler: s.router,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Printf("Server starting on port %s", port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
}

func (s *AWSCommandServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	s.wg.Wait()
	log.Println("Server stopped gracefully")
}
