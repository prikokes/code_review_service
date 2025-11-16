package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestResult struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalDuration      time.Duration
	AvgResponseTime    time.Duration
	MaxResponseTime    time.Duration
	ErrorRate          float64
}

func runLoadTest() *LoadTestResult {
	const (
		baseURL     = "http://localhost:8080"
		numRequests = 150
		concurrent  = 5
		targetRPS   = 5
	)

	var (
		totalRequests   int64
		successful      int64
		failed          int64
		totalDuration   int64
		maxResponseTime int64
		wg              sync.WaitGroup
	)

	semaphore := make(chan struct{}, concurrent)
	ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
	defer ticker.Stop()

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		<-ticker.C
		wg.Add(1)
		semaphore <- struct{}{}

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			start := time.Now()

			var url string
			switch reqNum % 3 {
			case 0:
				url = fmt.Sprintf("%s/team/get?team_name=backend", baseURL)
			case 1:
				url = fmt.Sprintf("%s/users/getReview?user_id=user_backend_1", baseURL)
			case 2:
				url = fmt.Sprintf("%s/team/get?team_name=frontend", baseURL)
			}

			resp, err := http.Get(url)
			duration := time.Since(start).Nanoseconds()

			atomic.AddInt64(&totalRequests, 1)
			atomic.AddInt64(&totalDuration, duration)

			if currentMax := atomic.LoadInt64(&maxResponseTime); duration > currentMax {
				atomic.CompareAndSwapInt64(&maxResponseTime, currentMax, duration)
			}

			if err != nil || resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failed, 1)
				if resp != nil {
					resp.Body.Close()
				}
				return
			}

			atomic.AddInt64(&successful, 1)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	result := &LoadTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successful,
		FailedRequests:     failed,
		TotalDuration:      totalTime,
	}

	if totalRequests > 0 {
		result.AvgResponseTime = time.Duration(totalDuration / totalRequests)
		result.MaxResponseTime = time.Duration(maxResponseTime)
		result.ErrorRate = float64(failed) / float64(totalRequests)
	}

	return result
}

func main() {
	fmt.Println("Starting load test...")
	result := runLoadTest()

	fmt.Printf("\n=== LOAD TEST RESULTS ===\n")
	fmt.Printf("Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("Successful: %d\n", result.SuccessfulRequests)
	fmt.Printf("Failed: %d\n", result.FailedRequests)
	fmt.Printf("Total Duration: %v\n", result.TotalDuration)
	fmt.Printf("Average Response Time: %v\n", result.AvgResponseTime)
	fmt.Printf("Max Response Time: %v\n", result.MaxResponseTime)
	fmt.Printf("Error Rate: %.4f%%\n", result.ErrorRate*100)

	if result.AvgResponseTime < 300*time.Millisecond {
		fmt.Printf("SLI Response Time: PASSED (< 300ms)\n")
	} else {
		fmt.Printf("SLI Response Time: FAILED\n")
	}

	if result.ErrorRate < 0.001 {
		fmt.Printf("SLI Success Rate: PASSED (> 99.9%%)\n")
	} else {
		fmt.Printf("SLI Success Rate: FAILED\n")
	}
}
