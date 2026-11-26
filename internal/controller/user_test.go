package controller

import (
	"community/internal/config"
	"community/internal/db"
	"community/internal/model"
	"community/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestUserControllerAuth(t *testing.T) {
	const password = "correct-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "existing", Password: string(hash)}
	user.ID = 1

	oldConfig := config.Cfg
	config.Cfg = &config.AppConfig{JWTSecret: "unit-test-only-key"}
	t.Cleanup(func() { config.Cfg = oldConfig })

	controller := &UserController{userService: service.UserService{}}
	queryFailure := errors.New("database query failed")
	tests := []struct {
		name      string
		handler   gin.HandlerFunc
		body      string
		queryErr  error
		createErr error
		wantCode  int
		wantError string
	}{
		{"注册时用户名已存在", controller.Register, `{"username":"existing","password":"correct-password"}`, nil, nil, http.StatusConflict, "注册失败：用户名已存在"},
		{"不存在的用户登录", controller.Login, `{"username":"missing","password":"correct-password"}`, gorm.ErrRecordNotFound, nil, http.StatusUnauthorized, "登录失败：用户名或密码错误"},
		{"登录时密码错误", controller.Login, `{"username":"existing","password":"wrong-password"}`, nil, nil, http.StatusUnauthorized, "登录失败：用户名或密码错误"},
		{"正常登录", controller.Login, `{"username":"existing","password":"correct-password"}`, nil, nil, http.StatusOK, ""},
		{"登录时查询失败", controller.Login, `{"username":"existing","password":"correct-password"}`, queryFailure, nil, http.StatusInternalServerError, "服务器内部错误"},
		{"注册时查询失败", controller.Register, `{"username":"existing","password":"correct-password"}`, queryFailure, nil, http.StatusInternalServerError, "服务器内部错误"},
		{"查询后发生用户名冲突", controller.Register, `{"username":"existing","password":"correct-password"}`, gorm.ErrRecordNotFound, &mysqlDriver.MySQLError{Number: 1062, Message: "Duplicate entry"}, http.StatusConflict, "注册失败：用户名已存在"},
		{"普通插入失败", controller.Register, `{"username":"newuser","password":"correct-password"}`, gorm.ErrRecordNotFound, errors.New("database insert failed"), http.StatusInternalServerError, "服务器内部错误"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB, err := gorm.Open(mysql.New(mysql.Config{
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				DisableAutomaticPing:   true,
				SkipDefaultTransaction: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := testDB.DB()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })

			// 模拟查询结果，保留服务层的密码校验和错误转换。
			err = testDB.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
				if tt.queryErr != nil {
					tx.AddError(tt.queryErr)
					return
				}
				*tx.Statement.Dest.(*model.User) = user
				tx.RowsAffected = 1
			})
			if err != nil {
				t.Fatal(err)
			}

			err = testDB.Callback().Create().Replace("gorm:create", func(tx *gorm.DB) {
				if tt.createErr != nil {
					tx.AddError(tt.createErr)
					return
				}
				tx.RowsAffected = 1
			})
			if err != nil {
				t.Fatal(err)
			}
			oldDB := db.DB
			db.DB = testDB
			t.Cleanup(func() { db.DB = oldDB })

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			t.Cleanup(func() { _ = c.Request.Body.Close() })

			tt.handler(c)

			if recorder.Code != tt.wantCode {
				t.Errorf("状态码 = %d，期望 = %d，响应 = %s", recorder.Code, tt.wantCode, recorder.Body.String())
			}
			if tt.wantError != "" {
				var body struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Fatalf("响应不是合法 JSON：%v；响应：%s", err, recorder.Body.String())
				}
				if body.Error != tt.wantError {
					t.Errorf("错误文案 = %q，期望 = %q", body.Error, tt.wantError)
				}
			}
		})
	}
}
