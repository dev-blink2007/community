package controller

import (
	"net/url"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name       string
		query      url.Values
		wantPage   int
		wantSize   int
		wantOffset int
		wantErr    bool
	}{
		{
			name: "正常第二页",
			query: url.Values{
				"page": []string{"2"},
				"size": []string{"5"},
			},
			wantPage:   2,
			wantSize:   5,
			wantOffset: 5,
			wantErr:    false,
		},
		{
			name:       "缺省使用默认值",
			query:      url.Values{},
			wantPage:   1,
			wantSize:   10,
			wantOffset: 0,
			wantErr:    false,
		},
		{
			name: "页码不是数字",
			query: url.Values{
				"page": []string{"abc"},
			},
			wantErr: true,
		},
		{
			name: "显式传入空页码",
			query: url.Values{
				"page": []string{""},
			},
			wantErr: true,
		},
		{
			name: "page不超过最大值",
			query: url.Values{
				"page": []string{"2001"},
				"size": []string{"50"},
			},
			wantPage:   2001,
			wantSize:   50,
			wantOffset: 100000,
			wantErr:    false,
		},
		{
			name: "page超过最大值",
			query: url.Values{
				"page": []string{"2002"},
				"size": []string{"50"},
			},
			wantErr: true,
		},
	}

	// 第三步：逐条取出用例，执行相同的检查逻辑。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, size, offset, err := ParsePagination(tt.query)

			if tt.wantErr {
				// 这个用例应该失败。
				if err == nil {
					t.Fatal("预期返回错误，实际没有错误")
				}
				return
			}

			// 这个用例应该成功。
			if err != nil {
				t.Fatalf("预期成功，实际错误：%v", err)
			}

			if page != tt.wantPage {
				t.Errorf("page：期望 %d，实际 %d", tt.wantPage, page)
			}

			if size != tt.wantSize {
				t.Errorf("size：期望 %d，实际 %d", tt.wantSize, size)
			}

			if offset != tt.wantOffset {
				t.Errorf(
					"offset：期望 %d，实际 %d",
					tt.wantOffset,
					offset,
				)
			}
		})
	}
}
