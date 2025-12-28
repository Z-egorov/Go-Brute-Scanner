package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Z-egorov/Go-Brute-Scanner/pkg/bruteforce"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/discovery"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/httpclient"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/wordlists"
)

// Option to configure scanner
type Option func(*types.Config)

// Scanner main interface
type Scanner interface {
	Discover(ctx context.Context) ([]types.Endpoint, error)
	Scan(ctx context.Context, methods []string, delay time.Duration) ([]types.ScanResult, error)
	ScanWithWordlist(ctx context.Context, wordlist []string, methods []string, concurrency int, delay time.Duration) ([]types.ScanResult, error)
	GetStats() types.Stats
	Stop() error
}

// scannerImpl implements scanner interface
type scannerImpl struct {
	config     types.Config
	client     *httpclient.Client
	discoverer *discovery.Crawler
	bf         bruteforce.Scanner
	wordlists  *wordlists.Common
	stats      types.Stats
	mu         sync.RWMutex
	cancelFunc context.CancelFunc
}

// New creates new scanner
func New(baseURL string, opts ...Option) (Scanner, error) {
	config := types.Config{
		BaseURL:      baseURL,
		Timeout:      10 * time.Second,
		UserAgent:    "GoBruteScanner/1.0",
		Workers:      5,
		MaxRedirects: 3,
		ScanDepth:    2,
		UseProxies:   false,
		ProxyRotate:  false,
		RateLimit:    10,
		Headers:      make(map[string]string),
		Cookies:      make(map[string]string),
		InsecureSSL:  false,
	}

	for _, opt := range opts {
		opt(&config)
	}

	client, err := httpclient.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	crawler := discovery.NewCrawler(client, config.ScanDepth)

	bfScanner := bruteforce.NewScanner(client)

	wl := wordlists.New()

	return &scannerImpl{
		config:     config,
		client:     client,
		discoverer: crawler,
		bf:         bfScanner,
		wordlists:  wl,
		stats: types.Stats{
			StartTime: time.Now(),
		},
	}, nil
}

// WithTimeout sets timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *types.Config) {
		c.Timeout = timeout
	}
}

// WithWorkers sets workers amount
func WithWorkers(workers int) Option {
	return func(c *types.Config) {
		c.Workers = workers
	}
}

// WithUserAgent sets User-Agent
func WithUserAgent(ua string) Option {
	return func(c *types.Config) {
		c.UserAgent = ua
	}
}

// WithScanDepth sets depth of scan
func WithScanDepth(depth int) Option {
	return func(c *types.Config) {
		c.ScanDepth = depth
	}
}

// WithProxies enables all proxies
func WithProxies(rotate bool) Option {
	return func(c *types.Config) {
		c.UseProxies = true
		c.ProxyRotate = rotate
	}
}

// WithProxyURLs adds proxy URLs
func WithProxyURLs(urls ...string) Option {
	return func(c *types.Config) {
		c.ProxyURLs = urls
	}
}

// Discover endpoints auto-detect
func (s *scannerImpl) Discover(ctx context.Context) ([]types.Endpoint, error) {
	s.mu.Lock()
	s.stats.DiscoveryStartTime = time.Now()
	s.mu.Unlock()

	endpoints, err := s.discoverer.Crawl(ctx, s.config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("crawling failed: %w", err)
	}

	s.mu.Lock()
	s.stats.TotalDiscovered = len(endpoints)
	s.stats.DiscoveryDuration = time.Since(s.stats.DiscoveryStartTime)
	s.mu.Unlock()

	return endpoints, nil
}

// Scan bruteforce scan
func (s *scannerImpl) Scan(ctx context.Context, methods []string, delay time.Duration) ([]types.ScanResult, error) {
	return s.ScanWithWordlist(ctx, s.wordlists.GetAll(), methods, s.config.Workers, delay)
}

// ScanWithWordlist scan with wordlist
func (s *scannerImpl) ScanWithWordlist(ctx context.Context, wordlist []string, methods []string, concurrency int, delay time.Duration) ([]types.ScanResult, error) {
	s.mu.Lock()
	s.stats.ScanStartTime = time.Now()
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFunc = cancel
	s.mu.Unlock()

	defer cancel()

	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "DELETE"}
	}

	bruteResults, err := s.bf.ScanWordlist(ctx, s.config.BaseURL, wordlist, methods, concurrency, delay)
	if err != nil {
		return nil, fmt.Errorf("brute force scan failed: %w", err)
	}

	scanResults := make([]types.ScanResult, len(bruteResults))
	for i, r := range bruteResults {
		scanResults[i] = types.ScanResult{
			URL:        r.URL,
			Method:     r.Method,
			StatusCode: r.StatusCode,
			Size:       r.Size,
			Headers:    r.Headers,
			Title:      r.Title,
			FoundVia:   "bruteforce",
			Timestamp:  r.Timestamp,
			Error:      r.Error,
		}
	}

	s.mu.Lock()
	s.stats.TotalRequests += len(bruteResults)
	for _, r := range bruteResults {
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			s.stats.Successful++
		} else if r.StatusCode >= 400 {
			s.stats.Failed++
		}
	}
	s.stats.ScanDuration = time.Since(s.stats.ScanStartTime)
	s.stats.Duration = time.Since(s.stats.StartTime)
	s.mu.Unlock()

	return scanResults, nil
}

// GetStats returns statistics
func (s *scannerImpl) GetStats() types.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// Stop stops scan
func (s *scannerImpl) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}

	return nil
}

// Reset resets statistics
func (s *scannerImpl) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats = types.Stats{
		StartTime: time.Now(),
	}
	s.discoverer.Clear()
}
