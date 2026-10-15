package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// 多个请求共用一个客户端，复用底层连接。
var agentHTTPClient = &http.Client{
	Timeout: 3 * time.Minute,
}

type AgentService struct{}

func (s *AgentService) Run(
	ctx context.Context,
	question string,
) ([]byte, int, error) {
	baseURL := strings.TrimSpace(os.Getenv("AGENT_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	body, err := json.Marshal(map[string]string{
		"question": question,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("编码 Agent 请求失败：%w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/agent/run",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("创建 Agent 请求失败：%w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := agentHTTPClient.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("请求 Agent 服务失败：%w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("读取 Agent 响应失败：%w", err)
	}

	if !json.Valid(responseBody) {
		return nil, 0, fmt.Errorf(
			"Agent 返回非 JSON 响应，HTTP 状态码：%d",
			response.StatusCode,
		)
	}

	return responseBody, response.StatusCode, nil
}
