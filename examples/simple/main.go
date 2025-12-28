package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Z-egorov/Go-Brute-Scanner/pkg/scanner"
)

func main() {
	fmt.Println("=== Go Brute Scanner - Simple Example ===")

	s, err := scanner.New(
		"http://localhost:8080",
		scanner.WithTimeout(10*time.Second),
		scanner.WithWorkers(5),
	)
	if err != nil {
		fmt.Printf("Failed to create scanner: %v\n", err)
		return
	}

	ctx := context.Background()

	fmt.Println("\n1. Auto-discovery phase:")
	fmt.Println("Scanning for endpoints...")

	endpoints, err := s.Discover(ctx)
	if err != nil {
		fmt.Printf("Discovery error: %v\n", err)
	} else {
		fmt.Printf("✅ Discovered %d endpoints\n", len(endpoints))

		for i, ep := range endpoints {
			if i >= 10 {
				fmt.Println("... and more")
				break
			}
			fmt.Printf("  • %s %s (via %s)\n", ep.Method, ep.URL, ep.Source)
		}
	}

	fmt.Println("\n2. Brute force phase:")
	fmt.Println("Testing common endpoints...")

	results, err := s.Scan(ctx, []string{"GET", "POST"}, 50*time.Millisecond)
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		return
	}

	fmt.Println("\n3. Results analysis:")

	successful := 0
	clientErrors := 0
	serverErrors := 0

	for _, result := range results {
		status := result.StatusCode
		switch {
		case status >= 200 && status < 300:
			successful++
			fmt.Printf("✅ [%d] %s %s\n", status, result.Method, result.URL)
		case status >= 400 && status < 500:
			clientErrors++
		case status >= 500:
			serverErrors++
		}
	}

	stats := s.GetStats()
	fmt.Printf("\n4. Statistics:\n")
	fmt.Printf("   • Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("   • Successful (2xx): %d\n", successful)
	fmt.Printf("   • Client errors (4xx): %d\n", clientErrors)
	fmt.Printf("   • Server errors (5xx): %d\n", serverErrors)
	fmt.Printf("   • Discovered endpoints: %d\n", stats.TotalDiscovered)
	fmt.Printf("   • Total duration: %v\n", stats.Duration)

	fmt.Println("\n✅ Scan completed successfully!")
}
