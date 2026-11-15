package utils

import (
	"community/internal/config"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseToken(t *testing.T) {
	const secret = "unit-test-only-key-0123456789abcdef"
	const wrongSecret = "other-test-only-key-fedcba9876543210"

	// 临时替换配置，测试结束后恢复。不要使用 t.Parallel()。
	oldConfig := config.Cfg
	config.Cfg = &config.AppConfig{JWTSecret: secret}
	t.Cleanup(func() {
		config.Cfg = oldConfig
	})

	tests := []struct {
		name       string
		method     jwt.SigningMethod
		signingKey string
		userID     uint
		expiration string
		wantErr    bool
	}{
		{"valid", jwt.SigningMethodHS256, secret, 1, "future", false},
		{"expired", jwt.SigningMethodHS256, secret, 1, "past", true},
		{"missing_exp", jwt.SigningMethodHS256, secret, 1, "missing", true},
		{"zero_user_id", jwt.SigningMethodHS256, secret, 0, "future", true},
		{"wrong_algorithm_HS512", jwt.SigningMethodHS512, secret, 1, "future", true},
		{"wrong_signing_key", jwt.SigningMethodHS256, wrongSecret, 1, "future", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := Claims{UserID: tt.userID}

			switch tt.expiration {
			case "future":
				claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
			case "past":
				claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
			case "missing":
				claims.ExpiresAt = nil
			default:
				t.Fatalf("测试用例的 expiration 配置错误：%q", tt.expiration)
			}

			token := jwt.NewWithClaims(tt.method, claims)
			tokenString, err := token.SignedString([]byte(tt.signingKey))
			if err != nil {
				t.Fatalf("构造测试凭证失败：%v", err)
			}

			got, err := ParseToken(tokenString)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("应拒绝此凭证，实际 err=nil，claims=%+v", got)
				}
				if got != nil {
					t.Fatalf("解析失败时应返回 nil claims，实际：%+v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("合法凭证应解析成功，实际错误：%v", err)
			}
			if got == nil || got.UserID != tt.userID {
				t.Fatalf(
					"解析身份不正确，期望 user_id=%d，实际：%+v",
					tt.userID,
					got,
				)
			}
		})
	}
}
