package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
)

// Client implements HTTP Client interface
type Client struct {
	client       *http.Client
	config       types.Config
	proxies      []*url.URL
	currentProxy int
	mu           sync.Mutex
}

// New creates client
func New(config types.Config) (*Client, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSSL,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	if config.MaxRedirects >= 0 {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}

	client := &Client{
		client: httpClient,
		config: config,
	}

	if len(config.ProxyURLs) > 0 {
		for _, proxyURL := range config.ProxyURLs {
			parsed, err := url.Parse(proxyURL)
			if err == nil {
				client.proxies = append(client.proxies, parsed)
			}
		}

		if len(client.proxies) > 0 {
			client.setProxy(client.proxies[0])
		}
	}

	return client, nil
}

// Do makes req
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" && c.config.UserAgent != "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}

	for k, v := range c.config.Headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	if len(c.config.Cookies) > 0 {
		for name, value := range c.config.Cookies {
			req.AddCookie(&http.Cookie{
				Name:  name,
				Value: value,
			})
		}
	}

	resp, err := c.client.Do(req)

	if err != nil && c.config.ProxyRotate && len(c.proxies) > 1 {
		c.mu.Lock()
		c.currentProxy = (c.currentProxy + 1) % len(c.proxies)
		c.setProxy(c.proxies[c.currentProxy])
		c.mu.Unlock()
	}

	return resp, err
}

// SetProxy sets proxy
func (c *Client) SetProxy(proxyURL string) error {
	if proxyURL == "" {
		c.client.Transport.(*http.Transport).Proxy = nil
		return nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}

	return c.setProxy(parsed)
}

// RotateProxy switches proxy
func (c *Client) RotateProxy() error {
	if len(c.proxies) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.currentProxy = (c.currentProxy + 1) % len(c.proxies)
	return c.setProxy(c.proxies[c.currentProxy])
}

// GetProxy returns current proxy
func (c *Client) GetProxy() string {
	if len(c.proxies) == 0 {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.currentProxy < len(c.proxies) {
		return c.proxies[c.currentProxy].String()
	}
	return ""
}

func (c *Client) setProxy(proxy *url.URL) error {
	c.client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxy)
	return nil
}
