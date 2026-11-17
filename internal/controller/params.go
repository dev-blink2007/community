package controller

import (
	"fmt"
	"net/url"
	"strconv"
)

func ParsePositiveID(param string) (uint, error) {
	ans, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("地址参数不合法:%s", err)
	}
	if ans <= 0 {
		return 0, fmt.Errorf("地址参数应该为正整数")
	}
	return uint(ans), nil
}

func ParsePagination(query url.Values) (page, size, offset int, err error) {
	if !query.Has("size") {
		size = 10
	} else {
		size, err = strconv.Atoi(query.Get("size"))
		if err != nil {
			return 0, 0, 0, fmt.Errorf("size参数不合法:%s", err)
		}
		if size <= 0 || size > 50 {
			return 0, 0, 0, fmt.Errorf("size 参数必须在 1～50 之间")
		}
	}
	if !query.Has("page") {
		page = 1
	} else {
		page, err = strconv.Atoi(query.Get("page"))
		if err != nil {
			return 0, 0, 0, fmt.Errorf("page参数不合法:%s", err)
		}
		if page <= 0 {
			return 0, 0, 0, fmt.Errorf("page参数应为正整数")
		}
		if page-1 > 100000/size {
			return 0, 0, 0, fmt.Errorf("page参数过大,超出读取范围")
		}
	}
	offset = (page - 1) * size
	return page, size, offset, nil
}
