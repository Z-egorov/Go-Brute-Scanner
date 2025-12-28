package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Z-egorov/Go-Brute-Scanner/pkg/output"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/scanner"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/wordlists"
)

func main() {
	fmt.Println("=== Go Brute Scanner - Advanced Example ===")

	fmt.Print("Enter target URL (default: http://localhost:8080): ")
	var targetURL string
	fmt.Scanln(&targetURL)
	if targetURL == "" {
		targetURL = "http://localhost:8080"
	}

	s, err := scanner.New(
		targetURL,
		scanner.WithTimeout(30*time.Second),
		scanner.WithWorkers(15),
		scanner.WithScanDepth(3),
		scanner.WithUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		scanner.WithProxies(false),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create scanner: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("\n🔍 Phase 1: Deep Discovery")
	fmt.Println("Crawling the website for endpoints...")

	startTime := time.Now()
	endpoints, err := s.Discover(ctx)
	if err != nil {
		fmt.Printf("⚠️ Discovery encountered errors: %v\n", err)
	}

	fmt.Printf("Discovered %d unique endpoints in %v\n",
		len(endpoints), time.Since(startTime))

	fmt.Println("\n⚡ Phase 2: Custom Brute Force")

	customWordlist := []string{
		// API endpoints
		"api/v1/users", "api/v1/products", "api/v1/orders",
		"api/v2/users", "api/v2/products", "api/v2/orders",
		"graphql", "gql", "rest", "soap",

		// Admin areas
		"admin", "admin/login", "admin/dashboard",
		"wp-admin", "wp-login.php", "administrator",

		// Common patterns
		"search", "search/users", "search/products",
		"upload", "upload/image", "upload/file",
		"download", "export", "import",

		// Health checks
		"health", "healthcheck", "status", "ping",
		"metrics", "info", "version",

		// Authentication
		"oauth2", "oauth2/authorize", "oauth2/token",
		"sso", "saml", "openid",
	}

	wl := wordlists.New()
	customWordlist = append(customWordlist, wl.GetAll()...)

	fmt.Printf("Testing %d paths with 4 HTTP methods...\n", len(customWordlist))

	results, err := s.ScanWithWordlist(
		ctx,
		customWordlist,
		[]string{"GET", "POST", "PUT", "DELETE"},
		10,
		100*time.Millisecond,
	)
	if err != nil {
		fmt.Printf("❌ Scan failed: %v\n", err)
		return
	}

	fmt.Println("\n📊 Phase 3: Results Analysis")

	successfulEndpoints := make(map[string][]types.ScanResult)
	var interestingEndpoints []types.ScanResult

	for _, result := range results {
		if result.StatusCode >= 200 && result.StatusCode < 300 {
			key := fmt.Sprintf("%s %s", result.Method, result.URL)
			successfulEndpoints[key] = append(successfulEndpoints[key], result)

			if isInterestingEndpoint(result) {
				interestingEndpoints = append(interestingEndpoints, result)
			}
		}
	}

	fmt.Printf("\n📋 Summary:\n")
	fmt.Printf("   • Total tested: %d\n", len(results))
	fmt.Printf("   • Successful endpoints: %d\n", len(successfulEndpoints))
	fmt.Printf("   • Interesting endpoints: %d\n", len(interestingEndpoints))

	if len(interestingEndpoints) > 0 {
		fmt.Println("\n🎯 Interesting endpoints found:")
		for i, ep := range interestingEndpoints {
			if i >= 10 {
				fmt.Printf("   ... and %d more\n", len(interestingEndpoints)-10)
				break
			}
			fmt.Printf("   • [%d] %s %s", ep.StatusCode, ep.Method, ep.URL)
			if ep.Title != "" {
				fmt.Printf(" - %s", ep.Title)
			}
			fmt.Println()
		}
	}

	fmt.Println("\n💾 Phase 4: Exporting Results")

	jsonFormatter := &output.JSONFormatter{Pretty: true}
	jsonData, _ := jsonFormatter.Format(interfaceSlice(results))

	jsonFile, err := os.Create("scan_results.json")
	if err == nil {
		jsonFile.WriteString(jsonData)
		jsonFile.Close()
		fmt.Println("   • JSON report saved to scan_results.json")
	}

	mdFormatter := &output.MarkdownFormatter{}
	mdData, _ := mdFormatter.Format(interfaceSlice(results))

	mdFile, err := os.Create("scan_results.md")
	if err == nil {
		mdFile.WriteString(mdData)
		mdFile.Close()
		fmt.Println("   • Markdown report saved to scan_results.md")
	}

	simpleFormatter := &output.SimpleFormatter{}
	txtData, _ := simpleFormatter.Format(interfaceSlice(results))

	txtFile, err := os.Create("endpoints.txt")
	if err == nil {
		txtFile.WriteString(txtData)
		txtFile.Close()
		fmt.Println("   • Simple list saved to endpoints.txt")
	}

	stats := s.GetStats()
	fmt.Println("\n📈 Detailed Statistics:")
	fmt.Printf("   • Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("   • Successful (2xx): %d\n", stats.Successful)
	fmt.Printf("   • Failed (4xx/5xx): %d\n", stats.Failed)
	fmt.Printf("   • Discovery time: %v\n", stats.DiscoveryDuration)
	fmt.Printf("   • Scan time: %v\n", stats.ScanDuration)
	fmt.Printf("   • Total time: %v\n", stats.Duration)
	fmt.Printf("   • Requests/sec: %.1f\n",
		float64(stats.TotalRequests)/stats.Duration.Seconds())

	fmt.Println("\n✅ Advanced scan completed!")
}

// isInterestingEndpoint finds interesting endpoints
func isInterestingEndpoint(result types.ScanResult) bool {
	url := result.URL

	// API endpoints
	if contains(url, "api", "rest", "graphql", "soap") {
		return true
	}

	// Admin endpoints
	if contains(url, "admin", "dashboard", "control", "manager") {
		return true
	}

	// Authentication
	if contains(url, "login", "auth", "oauth", "token", "register") {
		return true
	}

	// Data operations
	if contains(url, "upload", "download", "export", "import", "backup") {
		return true
	}

	// Configuration
	if contains(url, "config", "settings", "env", ".env", "secret") {
		return true
	}

	// Возвращает JSON или XML
	if result.Headers["Content-Type"] != "" {
		ct := result.Headers["Content-Type"]
		if contains(ct, "application/json", "application/xml", "text/xml") {
			return true
		}
	}

	return false
}

func contains(s string, substrings ...string) bool {
	lower := s
	for _, sub := range substrings {
		if len(sub) > 0 && len(lower) >= len(sub) {
			for i := 0; i <= len(lower)-len(sub); i++ {
				if lower[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func interfaceSlice(results []types.ScanResult) []interface{} {
	var slice []interface{}
	for _, r := range results {
		slice = append(slice, r)
	}
	return slice
}
