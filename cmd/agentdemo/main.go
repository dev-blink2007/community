package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	baseURL := strings.TrimSpace(os.Getenv("AGENT_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	payload := map[string]string{
		"question": "搜索孙颖莎相关帖子，读取正文，总结其中提到的2021年成绩。",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("编码请求失败：%w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		baseURL+"/agent/run",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("创建请求失败：%w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 3 * time.Minute,
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("请求 Agent 服务失败：%w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败：%w", err)
	}

	fmt.Println("HTTP 状态码：", response.StatusCode)
	fmt.Println("响应内容：", string(responseBody))

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"Agent 接口返回 HTTP %d，请查看上面的响应内容",
			response.StatusCode,
		)
	}

	return nil
}
