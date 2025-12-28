package discovery

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
	"golang.org/x/net/html"
)

// Crawler to find endpoints
type Crawler struct {
	client    types.HTTPClient
	maxDepth  int
	visited   map[string]bool
	baseURL   *url.URL
	mu        sync.RWMutex
	endpoints []types.Endpoint
}

// NewCrawler creates new crawler
func NewCrawler(client types.HTTPClient, maxDepth int) *Crawler {
	return &Crawler{
		client:    client,
		maxDepth:  maxDepth,
		visited:   make(map[string]bool),
		endpoints: make([]types.Endpoint, 0),
	}
}

// Crawl recursive scan
func (c *Crawler) Crawl(ctx context.Context, baseURL string) ([]types.Endpoint, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	c.baseURL = parsedURL

	return c.crawlRecursive(ctx, "/", 0)
}

// crawlRecursive recursive scan
func (c *Crawler) crawlRecursive(ctx context.Context, path string, depth int) ([]types.Endpoint, error) {
	if depth > c.maxDepth {
		return c.endpoints, nil
	}

	c.mu.Lock()
	if _, visited := c.visited[path]; visited {
		c.mu.Unlock()
		return c.endpoints, nil
	}
	c.visited[path] = true
	c.mu.Unlock()

	fullURL := c.resolvePath(path)

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return c.endpoints, err
	}
	req.Header.Set("User-Agent", "GoBruteScanner/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return c.endpoints, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.endpoints, err
	}

	bodyStr := string(body)

	c.extractLinks(bodyStr, path, depth)
	c.extractForms(bodyStr, path, depth)
	c.extractFromJS(bodyStr, path, depth)

	c.mu.Lock()
	c.endpoints = append(c.endpoints, types.Endpoint{
		URL:    fullURL,
		Method: "GET",
		Source: "direct",
		Depth:  depth,
	})
	c.mu.Unlock()

	var wg sync.WaitGroup
	c.mu.RLock()
	endpointsCopy := make([]types.Endpoint, len(c.endpoints))
	copy(endpointsCopy, c.endpoints)
	c.mu.RUnlock()

	for _, endpoint := range endpointsCopy {
		if endpoint.Depth == depth && endpoint.Source == "link" {
			wg.Add(1)
			go func(urlPath string, d int) {
				defer wg.Done()
				c.crawlRecursive(ctx, urlPath, d+1)
			}(endpoint.URL, depth)
		}
	}
	wg.Wait()

	return c.endpoints, nil
}

// GetEndpoints returns all found endpoints
func (c *Crawler) GetEndpoints() []types.Endpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.endpoints
}

// Clear clears crawler's status
func (c *Crawler) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.visited = make(map[string]bool)
	c.endpoints = make([]types.Endpoint, 0)
}

// resolvePath resolves a relative path to an absolute URL
func (c *Crawler) resolvePath(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}

	resolved := c.baseURL.ResolveReference(&url.URL{Path: strings.TrimPrefix(path, "/")})
	return resolved.String()
}

// extractLinks extracts links from html
func (c *Crawler) extractLinks(htmlContent, currentPath string, depth int) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		c.extractLinksWithTokenizer(htmlContent, currentPath, depth)
		return
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}

		normalized := c.normalizeURL(href, currentPath)
		if normalized == "" {
			return
		}

		c.mu.Lock()
		defer c.mu.Unlock()

		if _, visited := c.visited[normalized]; !visited {
			c.endpoints = append(c.endpoints, types.Endpoint{
				URL:    normalized,
				Method: "GET",
				Source: "link",
				Depth:  depth + 1,
				Metadata: map[string]interface{}{
					"text": s.Text(),
				},
			})
		}
	})
}

// extractLinksWithTokenizer alt method to extarct links
func (c *Crawler) extractLinksWithTokenizer(htmlContent, currentPath string, depth int) {
	tokenizer := html.NewTokenizer(strings.NewReader(htmlContent))

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return
		case html.StartTagToken, html.SelfClosingTagToken:
			t := tokenizer.Token()
			if t.Data == "a" {
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						href := attr.Val
						if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
							continue
						}

						normalized := c.normalizeURL(href, currentPath)
						if normalized == "" {
							continue
						}

						c.mu.Lock()
						if _, visited := c.visited[normalized]; !visited {
							c.endpoints = append(c.endpoints, types.Endpoint{
								URL:    normalized,
								Method: "GET",
								Source: "link",
								Depth:  depth + 1,
							})
						}
						c.mu.Unlock()
					}
				}
			}
		}
	}
}

// extractForms extracts url forms from html
func (c *Crawler) extractForms(htmlContent, currentPath string, depth int) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return
	}

	doc.Find("form").Each(func(i int, s *goquery.Selection) {
		action, _ := s.Attr("action")
		method, _ := s.Attr("method")
		if method == "" {
			method = "GET"
		}
		method = strings.ToUpper(method)

		if action == "" {
			action = currentPath
		}

		normalized := c.normalizeURL(action, currentPath)
		if normalized == "" {
			return
		}

		inputs := make(map[string]string)
		s.Find("input, textarea, select").Each(func(j int, input *goquery.Selection) {
			name, _ := input.Attr("name")
			typ, _ := input.Attr("type")
			value, _ := input.Attr("value")

			if name != "" && typ != "submit" && typ != "button" && typ != "reset" {
				if value == "" {
					switch typ {
					case "email":
						value = "test@example.com"
					case "password":
						value = "password123"
					case "number", "range":
						value = "1"
					case "checkbox", "radio":
						value = "on"
					default:
						value = "test"
					}
				}
				inputs[name] = value
			}
		})

		c.mu.Lock()
		c.endpoints = append(c.endpoints, types.Endpoint{
			URL:    normalized,
			Method: method,
			Source: "form",
			Depth:  depth + 1,
			Metadata: map[string]interface{}{
				"inputs": inputs,
			},
		})
		c.mu.Unlock()
	})
}

// extractFromJS extracts URL from js
func (c *Crawler) extractFromJS(jsContent, currentPath string, depth int) {
	patterns := []struct {
		regex *regexp.Regexp
	}{
		{regexp.MustCompile(`fetch\(['"]([^'"\s]+)['"]`)},
		{regexp.MustCompile(`fetch\(['"]([^'"\s]+)['"][^)]*method:\s*['"](GET|POST|PUT|DELETE|PATCH)['"]`)},

		{regexp.MustCompile(`axios\.(get|post|put|delete|patch)\(['"]([^'"\s]+)['"]`)},
		{regexp.MustCompile(`axios\([^)]*url:\s*['"]([^'"\s]+)['"]`)},

		{regexp.MustCompile(`\.open\(['"](GET|POST|PUT|DELETE|PATCH)['"][^,]*,['"]([^'"\s]+)['"]`)},

		{regexp.MustCompile(`\$\.(get|post|ajax)\([^)]*url:\s*['"]([^'"\s]+)['"]`)},

		{regexp.MustCompile(`window\.location\s*=\s*['"]([^'"\s]+)['"]`)},
		{regexp.MustCompile(`location\.href\s*=\s*['"]([^'"\s]+)['"]`)},

		{regexp.MustCompile(`['"](/[^'"\s]+)['"]`)},
	}

	for _, pattern := range patterns {
		matches := pattern.regex.FindAllStringSubmatch(jsContent, -1)
		for _, match := range matches {
			var path, method string

			if len(match) >= 3 {
				method = match[1]
				path = match[2]
			} else if len(match) >= 2 {
				path = match[1]
				method = "GET"
			}

			if path != "" {
				normalized := c.normalizeURL(path, currentPath)
				if normalized != "" {
					c.mu.Lock()
					c.endpoints = append(c.endpoints, types.Endpoint{
						URL:    normalized,
						Method: method,
						Source: "javascript",
						Depth:  depth + 1,
						Metadata: map[string]interface{}{
							"pattern": pattern.regex.String(),
						},
					})
					c.mu.Unlock()
				}
			}
		}
	}
}

// normalizeURL normalizes URL
func (c *Crawler) normalizeURL(href, currentPath string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		parsed, err := url.Parse(href)
		if err != nil || parsed.Host != c.baseURL.Host {
			return ""
		}
		return href
	}

	if strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
		return ""
	}

	var resolved string
	if strings.HasPrefix(href, "/") {
		resolved = c.baseURL.Scheme + "://" + c.baseURL.Host + href
	} else {
		currentURL := c.resolvePath(currentPath)
		parsedCurrent, err := url.Parse(currentURL)
		if err != nil {
			return ""
		}

		resolvedURL := parsedCurrent.ResolveReference(&url.URL{Path: href})
		resolved = resolvedURL.String()
	}

	return resolved
}
