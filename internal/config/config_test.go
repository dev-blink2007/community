package config

import (
	"os"
	"testing"
)

func TestLoadConfigJWTSecret(t *testing.T) {
	// 使用空临时目录，防止真实 .env 干扰测试。
	// t.Chdir 会在测试结束后恢复目录；你的 Go 1.26.3 支持。
	t.Chdir(t.TempDir())

	// 修改环境变量和全局配置的测试，不使用 t.Parallel()。
	tests := []struct {
		name    string
		value   string
		unset   bool
		wantErr bool
	}{
		{"valid", "unit-test-only-key-0123456789abcdef", false, false},
		{"missing", "", true, true},
		{"empty", "", false, true},
		{"whitespace", " \t\n ", false, true},
		{"public_default", "default-secret", false, true},
		{"padded_public_default", "  default-secret  ", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setenv 自动登记恢复操作。
			t.Setenv("JWT_SECRET", tt.value)

			// 区分“变量缺失”和“变量存在但为空”。
			if tt.unset {
				if err := os.Unsetenv("JWT_SECRET"); err != nil {
					t.Fatalf("准备缺失配置场景失败：%v", err)
				}
			}

			oldConfig := Cfg
			previous := &AppConfig{
				JWTSecret: "previous-test-config-0123456789abcdef",
			}
			Cfg = previous
			t.Cleanup(func() {
				Cfg = oldConfig
			})

			err := LoadConfig()

			if tt.wantErr {
				if err == nil {
					t.Fatal("此配置应被拒绝，但 LoadConfig 返回 nil")
				}
				if Cfg != previous {
					t.Error("校验失败后不应覆盖原来的全局 Cfg")
				}
				return
			}

			if err != nil {
				t.Fatalf("合法配置应加载成功，实际错误：%v", err)
			}
			if Cfg == nil {
				t.Fatal("加载成功后 Cfg 不应为 nil")
			}
			if Cfg.JWTSecret != tt.value {
				t.Error("加载成功后 JWTSecret 与传入配置不一致")
			}
		})
	}
}
