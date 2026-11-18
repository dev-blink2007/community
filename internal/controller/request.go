package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const maxBodyBytes int64 = 1 << 20

func BindJSONLimited(c *gin.Context, req any) error {
	limited := http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	defer limited.Close()
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	return c.ShouldBindJSON(req)
}
func ValidateText(content string, limitsize int, required bool) (string, error) {
	content = strings.TrimSpace(content)
	if required == true && content == "" {
		return "", fmt.Errorf("内容不应该为空")
	}
	contentlen := utf8.RuneCountInString(content)
	if contentlen > limitsize {
		return "", fmt.Errorf("内容大小超过限制,应小于%d个字符", limitsize)
	}
	return content, nil
}
func ValidatePassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if utf8.RuneCountInString(password) < 6 {
		return "", fmt.Errorf("密码应该大于6个字符")
	}

	if len(password) > 72 {
		return "", fmt.Errorf("密码应该小于72个字符")
	}
	return password, nil
}
