package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Z-egorov/Go-Brute-Scanner/pkg/scanner"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
)

func main() {
	fmt.Println("=== Go Brute Scanner - Custom Wordlist Example ===")

	targetBase := "http://api.example.com"

	customWordlist := generateDomainSpecificWordlist()

	fmt.Printf("Generated %d domain-specific paths\n\n", len(customWordlist))

	s, err := scanner.New(
		targetBase,
		scanner.WithTimeout(15*time.Second),
		scanner.WithWorkers(8),
		scanner.WithScanDepth(1),
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	fmt.Println("Starting targeted scan...")

	results, err := s.ScanWithWordlist(
		ctx,
		customWordlist,
		[]string{"GET", "POST", "PUT", "DELETE"},
		5,
		200*time.Millisecond,
	)
	if err != nil {
		panic(err)
	}

	categorizeAndPrintResults(results)
}

func generateDomainSpecificWordlist() []string {
	var wordlist []string

	basePatterns := []string{
		// Products
		"products", "products/{id}", "products/search",
		"products/category/{category}", "products/{id}/reviews",
		"products/{id}/images", "products/{id}/variants",

		// Users
		"users", "users/{id}", "users/{id}/profile",
		"users/{id}/orders", "users/{id}/addresses",
		"users/{id}/payment-methods",

		// Orders
		"orders", "orders/{id}", "orders/{id}/items",
		"orders/{id}/status", "orders/{id}/invoice",

		// Cart
		"cart", "cart/items", "cart/{id}",

		// Payments
		"payments", "payments/{id}", "payments/methods",
		"payments/webhook",

		// Search
		"search/products", "search/users", "search/orders",

		// Admin
		"admin/dashboard", "admin/products", "admin/orders",
		"admin/users", "admin/analytics", "admin/settings",

		// Analytics
		"analytics/sales", "analytics/users", "analytics/products",
		"analytics/reports",

		// Reports
		"reports/sales", "reports/users", "reports/inventory",

		// Inventory
		"inventory", "inventory/items", "inventory/low-stock",
		"inventory/warehouses",
	}

	versions := []string{"", "/v1", "/v2", "/v3", "/api/v1", "/api/v2"}

	for _, version := range versions {
		for _, pattern := range basePatterns {
			path := version
			if version != "" && !strings.HasPrefix(pattern, "/") {
				path += "/"
			}
			path += pattern

			wordlist = append(wordlist, path)

			if strings.Contains(path, "{id}") {
				wordlist = append(wordlist,
					strings.Replace(path, "{id}", "1", -1),
					strings.Replace(path, "{id}", "123", -1),
					strings.Replace(path, "{id}", "test", -1),
				)
			}

			if strings.Contains(path, "{category}") {
				wordlist = append(wordlist,
					strings.Replace(path, "{category}", "electronics", -1),
					strings.Replace(path, "{category}", "clothing", -1),
					strings.Replace(path, "{category}", "books", -1),
				)
			}
		}
	}

	uniquePaths := make(map[string]bool)
	for _, path := range wordlist {
		uniquePaths[strings.Trim(path, "/")] = true
	}

	result := make([]string, 0, len(uniquePaths))
	for path := range uniquePaths {
		if path != "" {
			result = append(result, path)
		}
	}

	return result
}

func categorizeAndPrintResults(results []types.ScanResult) {
	categories := map[string][]types.ScanResult{
		"products":  {},
		"users":     {},
		"orders":    {},
		"admin":     {},
		"auth":      {},
		"search":    {},
		"analytics": {},
		"misc":      {},
	}

	for _, result := range results {
		url := strings.ToLower(result.URL)

		switch {
		case strings.Contains(url, "product"):
			categories["products"] = append(categories["products"], result)
		case strings.Contains(url, "user"):
			categories["users"] = append(categories["users"], result)
		case strings.Contains(url, "order"):
			categories["orders"] = append(categories["orders"], result)
		case strings.Contains(url, "admin"):
			categories["admin"] = append(categories["admin"], result)
		case strings.Contains(url, "auth") || strings.Contains(url, "login") || strings.Contains(url, "register"):
			categories["auth"] = append(categories["auth"], result)
		case strings.Contains(url, "search"):
			categories["search"] = append(categories["search"], result)
		case strings.Contains(url, "analytic"):
			categories["analytics"] = append(categories["analytics"], result)
		default:
			categories["misc"] = append(categories["misc"], result)
		}
	}

	totalSuccessful := 0

	for category, items := range categories {
		if len(items) > 0 {
			successful := 0
			for _, item := range items {
				if item.StatusCode >= 200 && item.StatusCode < 300 {
					successful++
					totalSuccessful++
				}
			}

			fmt.Printf("%s: %d/%d successful\n",
				strings.Title(category), successful, len(items))

			shown := 0
			for _, item := range items {
				if item.StatusCode >= 200 && item.StatusCode < 300 && shown < 3 {
					fmt.Printf("  • %s %s\n", item.Method, item.URL)
					shown++
				}
			}
			if successful > 3 {
				fmt.Printf("  ... and %d more\n", successful-3)
			}
			fmt.Println()
		}
	}

	fmt.Printf("Total successful endpoints: %d/%d\n",
		totalSuccessful, len(results))
}
