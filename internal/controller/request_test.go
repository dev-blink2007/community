package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"空密码", "", true},
		{"不足6字符", strings.Repeat("a", 5), true},
		{"恰好6字符", strings.Repeat("a", 6), false},
		{"恰好72字节", strings.Repeat("a", 72), false},
		{"超过72字节", strings.Repeat("a", 73), true},
		{"中文不足6字符", strings.Repeat("中", 5), true},
		{"中文恰好6字符", strings.Repeat("中", 6), false},
		{"中文恰好72字节", strings.Repeat("中", 24), false},
		{"中文超过72字节", strings.Repeat("中", 25), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePassword(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"error = %v，wantErr = %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				if got != "" {
					t.Errorf("校验失败时应该返回空字符串")
				}
				return
			}

			if got != tt.input {
				t.Errorf("校验成功时不应该修改原始密码")
			}
		})
	}
}

func TestValidateText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		limit    int
		required bool
		wantErr  bool
	}{
		{"必填内容为空", "", 200, true, true},
		{"必填内容全为空白", " \t\n ", 200, true, true},
		{"非必填内容允许为空", "", 200, false, false},

		{"标题恰好200字符", strings.Repeat("a", 200), 200, true, false},
		{"标题超过200字符", strings.Repeat("a", 201), 200, true, true},
		{"中文按字符而非字节计数", strings.Repeat("中", 200), 200, true, false},
		{"中文超过字符上限", strings.Repeat("中", 201), 200, true, true},

		{"正文恰好达到上限", strings.Repeat("a", 20000), 20000, true, false},
		{"正文超过上限", strings.Repeat("a", 20001), 20000, true, true},
		{"评论恰好达到上限", strings.Repeat("a", 2000), 2000, true, false},
		{"评论超过上限", strings.Repeat("a", 2001), 2000, true, true},

		{"可选文本仍受长度限制", strings.Repeat("a", 201), 200, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateText(tt.input, tt.limit, tt.required)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"error = %v，wantErr = %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr {
				if got != "" {
					t.Errorf("校验失败时应该返回空字符串")
				}
				return
			}

			if got != tt.input {
				t.Errorf("校验成功时不应该修改输入文本")
			}
		})
	}
}

func TestBindJSONLimited(t *testing.T) {
	const limit = 1 << 20 // 1 MiB = 1048576 字节

	const prefix = `{"text":"`
	const suffix = `"}`

	// 生成总大小恰好为 size 字节的合法 JSON。
	makeBody := func(size int) string {
		contentSize := size - len(prefix) - len(suffix)
		return prefix + strings.Repeat("a", contentSize) + suffix
	}

	smallBody := `{"text":"hello"}`

	tests := []struct {
		name         string
		body         string
		wantErr      bool
		wantTooLarge bool
		wantText     string
	}{
		{
			name:     "正常JSON并填入结构体",
			body:     smallBody,
			wantText: "hello",
		},
		{
			name:     "小于上限1字节",
			body:     makeBody(limit - 1),
			wantText: strings.Repeat("a", limit-1-len(prefix)-len(suffix)),
		},
		{
			name:     "恰好达到上限",
			body:     makeBody(limit),
			wantText: strings.Repeat("a", limit-len(prefix)-len(suffix)),
		},
		{
			name:         "超过上限1字节",
			body:         makeBody(limit + 1),
			wantErr:      true,
			wantTooLarge: true,
		},
		{
			// 即使前面的 JSON 合法，也必须检查完整请求体大小。
			name: "合法JSON后追加空白导致整体超限",
			body: smallBody +
				strings.Repeat(" ", limit+1-len(smallBody)),
			wantErr:      true,
			wantTooLarge: true,
		},
		{
			name:    "空请求体",
			body:    "",
			wantErr: true,
		},
		{
			name:    "JSON语法错误",
			body:    `{"text":`,
			wantErr: true,
		},
		{
			name:    "字段类型错误",
			body:    `{"text":123}`,
			wantErr: true,
		},
		{
			name:    "保留binding必填校验",
			body:    `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			c.Request = httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(tt.body),
			)
			c.Request.Header.Set("Content-Type", "application/json")

			t.Cleanup(func() {
				_ = c.Request.Body.Close()
			})

			var req struct {
				Text string `json:"text" binding:"required"`
			}

			err := BindJSONLimited(c, &req)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"error = %v，wantErr = %v",
					err,
					tt.wantErr,
				)
			}

			// 确认超限错误的具体类型，避免把其他错误误判为超限。
			var sizeErr *http.MaxBytesError
			gotTooLarge := errors.As(err, &sizeErr)

			if gotTooLarge != tt.wantTooLarge {
				t.Fatalf(
					"超限错误 = %v，期望 = %v，实际错误 = %v",
					gotTooLarge,
					tt.wantTooLarge,
					err,
				)
			}

			if gotTooLarge && sizeErr.Limit != int64(limit) {
				t.Errorf(
					"字节上限 = %d，期望 = %d",
					sizeErr.Limit,
					limit,
				)
			}

			if !tt.wantErr && req.Text != tt.wantText {
				// 避免在失败时打印接近 1 MiB 的文本。
				t.Errorf(
					"解析后的文本与预期不一致：实际 %d 字节，期望 %d 字节",
					len(req.Text),
					len(tt.wantText),
				)
			}
		})
	}
}
