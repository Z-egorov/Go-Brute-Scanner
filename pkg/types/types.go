package types

import (
	"net/http"
	"time"
)

// HTTPClient http client interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	SetProxy(proxyURL string) error
	RotateProxy() error
	GetProxy() string
}

// Endpoint found endpoint
type Endpoint struct {
	URL      string                 `json:"url"`
	Method   string                 `json:"method"`
	Source   string                 `json:"source"`
	Depth    int                    `json:"depth"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ScanResult scan result
type ScanResult struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	StatusCode int               `json:"status_code"`
	Size       int               `json:"size"`
	Body       string            `json:"body,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Title      string            `json:"title,omitempty"`
	FoundVia   string            `json:"found_via"`
	Timestamp  time.Time         `json:"timestamp"`
	Error      string            `json:"error,omitempty"`
}

// Stats scan statistics
type Stats struct {
	TotalRequests      int           `json:"total_requests"`
	Successful         int           `json:"successful"`
	Failed             int           `json:"failed"`
	TotalDiscovered    int           `json:"total_discovered"`
	StartTime          time.Time     `json:"start_time"`
	DiscoveryStartTime time.Time     `json:"discovery_start_time,omitempty"`
	ScanStartTime      time.Time     `json:"scan_start_time,omitempty"`
	Duration           time.Duration `json:"duration"`
	DiscoveryDuration  time.Duration `json:"discovery_duration,omitempty"`
	ScanDuration       time.Duration `json:"scan_duration,omitempty"`
}

// Config scan cfg
type Config struct {
	BaseURL      string            `json:"base_url"`
	Timeout      time.Duration     `json:"timeout"`
	UserAgent    string            `json:"user_agent"`
	Workers      int               `json:"workers"`
	MaxRedirects int               `json:"max_redirects"`
	ScanDepth    int               `json:"scan_depth"`
	UseProxies   bool              `json:"use_proxies"`
	ProxyRotate  bool              `json:"proxy_rotate"`
	RateLimit    int               `json:"rate_limit"`
	Headers      map[string]string `json:"headers"`
	Cookies      map[string]string `json:"cookies"`
	InsecureSSL  bool              `json:"insecure_ssl"`
	ProxyURLs    []string          `json:"proxy_urls"`
}

// AuthConfig auth cfg
type AuthConfig struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
	Header   string `json:"header,omitempty"`
}

// BruteResult bruteforcer result
type BruteResult struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	StatusCode int               `json:"status_code"`
	Size       int               `json:"size"`
	Body       string            `json:"body,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Title      string            `json:"title,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Error      string            `json:"error,omitempty"`
}
