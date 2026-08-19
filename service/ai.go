package service

import (
	"bytes"
	"community/config"
	"community/redis"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const deepseekURL = "https://api.deepseek.com/chat/completions"

type deepseekRequest struct {
	Model    string        `json:"model"`
	Messages []deepseekMsg `json:"messages"`
	Stream   bool          `json:"stream"`
}

type deepseekMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// 调用API，返回ai的回复和token消耗
func CallDeepSeekAPI(ctx context.Context, prompt string, postid uint) (string, int, error) {
	summarykey := fmt.Sprintf("ai:summary:%d", postid)
	tokenkey := fmt.Sprintf("ai:summary:token:%d", postid)
	pipe := redis.RDB.Pipeline()
	contentcmd := pipe.Get(ctx, summarykey)
	tokencmd := pipe.Get(ctx, tokenkey)
	pipe.Exec(ctx)
	content, err1 := contentcmd.Result()
	token, err2 := tokencmd.Int()
	if err1 == nil && err2 == nil {
		return content, token, nil
	}
	//构造请求体
	reqBody := deepseekRequest{
		Model: "deepseek-chat",
		Messages: []deepseekMsg{
			{Role: "system", Content: "你是一个文本总结助手,只返回纯文本,不超过100字,不要加任何前缀或后缀"},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("创建请求失败：%v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", deepseekURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", 0, fmt.Errorf("创建请求体失败：%w", err)
	}

	apiKey := config.Cfg.DeepSeekAPIKey
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	//发送请求
	client := &http.Client{
		Timeout: 10 * time.Second, //额外兜底超时
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("请求AI失败:%w", err)
	}
	defer resp.Body.Close()
	//读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("读取响应体失败：%w", err)
	}

	//解析响应
	var aiResp deepseekResponse
	if err := json.Unmarshal(body, &aiResp); err != nil {
		return "", 0, fmt.Errorf("解析响应体失败：%w", err)
	}
	if len(aiResp.Choices) == 0 {
		return "", 0, fmt.Errorf("ai 返回为空")
	}
	content = aiResp.Choices[0].Message.Content
	token = aiResp.Usage.TotalTokens
	err1 = redis.RDB.Set(ctx, summarykey, content, 24*time.Hour).Err()
	err2 = redis.RDB.Set(ctx, tokenkey, token, 24*time.Hour).Err()
	if err1 != nil || err2 != nil {
		log.Printf("帖子 %d ai结果缓存失败", postid)
	}
	return content, token, nil
}
