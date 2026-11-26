package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"community/internal/db"
	"community/internal/model"
	myredis "community/internal/redis"

	"github.com/alicebob/miniredis/v2"
	redisClient "github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 这个 Hook 只用于测试：让指定 Redis 操作返回人为设置的错误。
// 其他操作仍交给临时 Redis 执行，包括真实运行解锁 Lua 脚本。
type postCacheFailureHook struct {
	operation string
	cacheKey  string
	lockKey   string
	err       error
}

func (h postCacheFailureHook) BeforeProcess(
	ctx context.Context,
	cmd redisClient.Cmder,
) (context.Context, error) {
	args := cmd.Args()
	if len(args) < 2 {
		return ctx, nil
	}

	key, _ := args[1].(string)
	name := cmd.Name()

	switch h.operation {
	case "get":
		if name == "get" && key == h.cacheKey {
			return ctx, h.err
		}
	case "lock":
		// 带过期时间的 SetNX 通常实际发送 SET ... NX。
		if (name == "set" || name == "setnx") && key == h.lockKey {
			return ctx, h.err
		}
	case "fill":
		if name == "set" && key == h.cacheKey {
			return ctx, h.err
		}
	}

	return ctx, nil
}

func (h postCacheFailureHook) AfterProcess(
	context.Context,
	redisClient.Cmder,
) error {
	return nil
}

func (h postCacheFailureHook) BeforeProcessPipeline(
	ctx context.Context,
	_ []redisClient.Cmder,
) (context.Context, error) {
	return ctx, nil
}

func (h postCacheFailureHook) AfterProcessPipeline(
	context.Context,
	[]redisClient.Cmder,
) error {
	return nil
}

func TestGetPostByIDWithLock(t *testing.T) {
	// 本测试会替换全局 db.DB 和 myredis.RDB，不要加 t.Parallel()。
	const (
		postID   = uint(900001)
		cacheKey = "post:900001"
		lockKey  = "post:900001:lock"
		ownerB   = "another-request-token"
	)

	expected := model.Post{
		Title:   "缓存测试帖子",
		Content: "这条数据只存在于测试中",
	}
	expected.ID = postID

	cachedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}

	redisFailure := errors.New("simulated redis failure")
	queryFailure := errors.New("simulated database failure")

	tests := []struct {
		name              string
		cacheHit          bool
		lockHeld          bool
		cancelBefore      bool
		cancelDuringQuery bool
		shortParent       bool
		failOperation     string
		queryErr          error
		wantErr           error
		wantQueries       int
		wantCache         bool
		checkOwnTimeout   bool
	}{
		{
			name:      "缓存命中不查询数据库",
			cacheHit:  true,
			wantCache: true,
		},
		{
			name:        "缓存未命中后查询回填并释放锁",
			wantQueries: 1,
			wantCache:   true,
		},
		{
			name:        "帖子不存在仍释放锁",
			queryErr:    gorm.ErrRecordNotFound,
			wantErr:     ErrPostNotFound,
			wantQueries: 1,
		},
		{
			name:        "数据库查询失败仍释放锁",
			queryErr:    queryFailure,
			wantErr:     queryFailure,
			wantQueries: 1,
		},
		{
			name:            "锁被占用时由函数自身两秒预算结束",
			lockHeld:        true,
			wantErr:         context.DeadlineExceeded,
			checkOwnTimeout: true,
		},
		{
			name:        "父Context更早到期",
			lockHeld:    true,
			shortParent: true,
			wantErr:     context.DeadlineExceeded,
		},
		{
			name:         "调用前已经取消",
			cancelBefore: true,
			wantErr:      context.Canceled,
		},
		{
			name:              "查询期间取消后仍能释放锁",
			cancelDuringQuery: true,
			wantErr:           context.Canceled,
			wantQueries:       1,
		},
		{
			name:          "Redis读取失败不查询数据库",
			failOperation: "get",
			wantErr:       redisFailure,
		},
		{
			name:          "Redis抢锁失败不查询数据库",
			failOperation: "lock",
			wantErr:       redisFailure,
		},
		{
			name:          "缓存回填失败仍返回帖子并释放锁",
			failOperation: "fill",
			wantQueries:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每个用例启动自己的临时 Redis，测试结束自动关闭。
			mr := miniredis.RunT(t)

			client := redisClient.NewClient(&redisClient.Options{
				Addr:       mr.Addr(),
				MaxRetries: -1,
			})

			oldRedis := myredis.RDB
			myredis.RDB = client
			t.Cleanup(func() {
				myredis.RDB = oldRedis
				_ = client.Close()
			})

			if tt.cacheHit {
				if err := mr.Set(cacheKey, string(cachedJSON)); err != nil {
					t.Fatal(err)
				}
			}

			if tt.lockHeld {
				if err := mr.Set(lockKey, ownerB); err != nil {
					t.Fatal(err)
				}
				mr.SetTTL(lockKey, 10*time.Second)
			}

			if tt.failOperation != "" {
				client.AddHook(postCacheFailureHook{
					operation: tt.failOperation,
					cacheKey:  cacheKey,
					lockKey:   lockKey,
					err:       redisFailure,
				})
			}

			// 外层五秒只是测试兜底。
			// 生产函数自己的两秒上限应该更早结束等待。
			parentBudget := 5 * time.Second
			if tt.shortParent {
				parentBudget = 150 * time.Millisecond
			}

			ctx, cancel := context.WithTimeout(
				context.Background(), parentBudget,
			)
			defer cancel()

			if tt.cancelBefore {
				cancel()
			}

			testDB, err := gorm.Open(mysql.New(mysql.Config{
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				DisableAutomaticPing: true,
				Logger:               logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				t.Fatal(err)
			}

			sqlDB, err := testDB.DB()
			if err != nil {
				t.Fatal(err)
			}

			oldDB := db.DB
			db.DB = testDB
			t.Cleanup(func() {
				db.DB = oldDB
				_ = sqlDB.Close()
			})

			// 本轮只测试帖子查询和缓存流程，不测试 User 关联预加载。
			// 去掉关联查询，避免测试尝试连接真实 MySQL。
			if err := testDB.Callback().Query().
				Remove("gorm:preload"); err != nil {
				t.Fatal(err)
			}

			queryCount := 0
			err = testDB.Callback().Query().
				Replace("gorm:query", func(tx *gorm.DB) {
					queryCount++

					// 检查 Service 是否确实通过 WithContext 传入 Context。
					if _, ok := tx.Statement.Context.Deadline(); !ok {
						t.Error("数据库查询未收到带截止时间的 Context")
					}

					if tt.cancelDuringQuery {
						cancel()
					}

					if err := tx.Statement.Context.Err(); err != nil {
						tx.AddError(err)
						return
					}

					if tt.queryErr != nil {
						tx.AddError(tt.queryErr)
						return
					}

					post, ok := tx.Statement.Dest.(*model.Post)
					if !ok {
						t.Errorf(
							"查询目标类型错误：%T",
							tx.Statement.Dest,
						)
						tx.AddError(errors.New("unexpected query destination"))
						return
					}

					*post = expected
					tx.RowsAffected = 1
				})
			if err != nil {
				t.Fatal(err)
			}

			svc := &PostService{}
			start := time.Now()
			got, err := svc.GetPostByIDWithLock(ctx, postID)
			elapsed := time.Since(start)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("错误 = %v，期望包含 %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("失败时应返回 nil 帖子，实际为 %+v", got)
				}
			} else {
				if err != nil {
					t.Fatalf("不应返回错误：%v", err)
				}
				if got == nil {
					t.Fatal("成功时返回了 nil 帖子")
				}
				if got.ID != expected.ID ||
					got.Title != expected.Title ||
					got.Content != expected.Content {
					t.Errorf("返回帖子不符合预期：%+v", got)
				}
			}

			if queryCount != tt.wantQueries {
				t.Errorf(
					"数据库查询次数 = %d，期望 %d",
					queryCount, tt.wantQueries,
				)
			}

			// 函数已经返回，defer 应当已经执行完毕。
			if tt.lockHeld {
				value, err := mr.Get(lockKey)
				if err != nil || value != ownerB {
					t.Errorf(
						"不能改变其他请求的锁：value=%q，err=%v",
						value, err,
					)
				}
			} else if mr.Exists(lockKey) {
				t.Error("函数返回后锁仍存在，检查 defer 和 Lua 脚本")
			}

			if tt.wantCache {
				value, err := mr.Get(cacheKey)
				if err != nil {
					t.Fatalf("期望缓存存在：%v", err)
				}
				var cached model.Post
				if err := json.Unmarshal([]byte(value), &cached); err != nil {
					t.Fatalf("缓存不是合法帖子 JSON：%v", err)
				}
				if cached.ID != expected.ID ||
					cached.Title != expected.Title ||
					cached.Content != expected.Content {
					t.Errorf("缓存内容不符合预期：%+v", cached)
				}
			} else if mr.Exists(cacheKey) {
				t.Error("本用例不应写入缓存")
			}

			// 给调度留余量，不要求毫秒级精确。
			if tt.checkOwnTimeout &&
				(elapsed < 1500*time.Millisecond ||
					elapsed > 4*time.Second) {
				t.Errorf("两秒预算未按预期生效，实际耗时 %v", elapsed)
			}

			if tt.shortParent && elapsed > time.Second {
				t.Errorf("没有及时响应父 Context 超时，耗时 %v", elapsed)
			}
		})
	}
}
