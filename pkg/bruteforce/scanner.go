package bruteforce

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
)

// Scanner bruteforcer interface
type Scanner interface {
	ScanPath(ctx context.Context, url string, methods []string, delay time.Duration) ([]types.BruteResult, error)
	ScanWordlist(ctx context.Context, baseURL string, wordlist []string, methods []string, concurrency int, delay time.Duration) ([]types.BruteResult, error)
}

// scannerImpl implements Scanner interface
type scannerImpl struct {
	client types.HTTPClient
}

// NewScanner creates bruteforcer
func NewScanner(client types.HTTPClient) Scanner {
	return &scannerImpl{
		client: client,
	}
}

// ScanPath scans path with some methods
func (s *scannerImpl) ScanPath(ctx context.Context, url string, methods []string, delay time.Duration) ([]types.BruteResult, error) {
	var results []types.BruteResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, method := range methods {
		wg.Add(1)

		go func(m string) {
			defer wg.Done()

			result := s.testEndpoint(ctx, url, m)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()

			time.Sleep(delay)
		}(method)
	}

	wg.Wait()
	return results, nil
}

// ScanWordlist scans path list
func (s *scannerImpl) ScanWordlist(ctx context.Context, baseURL string, wordlist []string, methods []string, concurrency int, delay time.Duration) ([]types.BruteResult, error) {
	// Создаем канал задач
	tasks := make(chan task, len(wordlist)*len(methods))
	for _, path := range wordlist {
		fullURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
		for _, method := range methods {
			tasks <- task{
				url:    fullURL,
				method: method,
			}
		}
	}
	close(tasks)

	var results []types.BruteResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for task := range tasks {
				select {
				case <-ctx.Done():
					return
				default:
					result := s.testEndpoint(ctx, task.url, task.method)

					mu.Lock()
					results = append(results, result)
					mu.Unlock()

					time.Sleep(delay)
				}
			}
		}(i)
	}

	wg.Wait()
	return results, nil
}

// testEndpoint tests endpoint
func (s *scannerImpl) testEndpoint(ctx context.Context, url, method string) types.BruteResult {
	result := types.BruteResult{
		URL:       url,
		Method:    method,
		Timestamp: time.Now(),
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("User-Agent", "GoBruteScanner/1.0")
	req.Header.Set("Accept", "*/*")
	if method == "POST" || method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Headers = make(map[string]string)

	for k, v := range resp.Header {
		if len(v) > 0 {
			result.Headers[k] = v[0]
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read body: %v", err)
		return result
	}

	result.Body = string(body)
	result.Size = len(body)

	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err == nil {
			result.Title = doc.Find("title").Text()
		}
	}

	return result
}

type task struct {
	url    string
	method string
}
