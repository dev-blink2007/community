package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	//可调参数
	url := "http://localhost:8080/api/posts"
	//请求数
	total := 1000
	//并发数
	concurrency := 50

	var wg sync.WaitGroup                   //计时器，等待所有请求完成
	var success, fail int64                 //原子计数器，记录成功/失败数
	sem := make(chan struct{}, concurrency) //信号量，控制并发数

	start := time.Now() //记录开始时间

	for i := 0; i < total; i++ {
		wg.Add(1)         //每发一个请求，计数器加一
		sem <- struct{}{} //占用一个并发槽，如果加满了就阻塞等待

		go func() {
			defer wg.Done()          //请求完成后，计数器减一
			defer func() { <-sem }() //释放一个并发槽

			resp, err := http.Get(url) //发送get请求
			if err != nil {
				atomic.AddInt64(&fail, 1)
				return
			}
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body) //读取并丢弃响应体（节省内存）
			if resp.StatusCode == 200 {
				atomic.AddInt64(&success, 1)
			} else {
				atomic.AddInt64(&fail, 1)
			}

		}()
	}
	wg.Wait()                    //阻塞，直到所有请求完成
	elapsed := time.Since(start) //计算总耗时

	fmt.Println("========== 压测报告 ==========")
	fmt.Printf("接口: %s\n", url)
	fmt.Printf("总请求数: %d\n", total)
	fmt.Printf("并发数: %d\n", concurrency)
	fmt.Printf("成功: %d\n", success)
	fmt.Printf("失败: %d\n", fail)
	fmt.Printf("总耗时: %v\n", elapsed)
	fmt.Printf("QPS: %.2f\n", float64(total)/elapsed.Seconds())
}
