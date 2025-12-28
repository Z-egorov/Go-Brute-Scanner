package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Z-egorov/Go-Brute-Scanner/pkg/output"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/scanner"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/wordlists"
)

func main() {
	// Парсим аргументы командной строки
	var (
		url        = flag.String("url", "", "Target URL (required)")
		workers    = flag.Int("workers", 10, "Number of concurrent workers")
		timeout    = flag.Int("timeout", 30, "Timeout in seconds")
		depth      = flag.Int("depth", 2, "Crawl depth")
		delay      = flag.Int("delay", 100, "Delay between requests in ms")
		methods    = flag.String("methods", "GET,POST,PUT,DELETE", "HTTP methods to test")
		outputFile = flag.String("output", "results.json", "Output file")
		format     = flag.String("format", "json", "Output format (json, md, txt)")
		discover   = flag.Bool("discover", true, "Enable auto-discovery")
		brute      = flag.Bool("brute", true, "Enable brute force")
		quiet      = flag.Bool("quiet", false, "Quiet mode (only results)")
		wordlist   = flag.String("wordlist", "", "Custom wordlist file (one per line)")
		proxies    = flag.String("proxies", "", "Proxy list file (one per line)")
	)

	flag.Parse()

	// Проверяем обязательные параметры
	if *url == "" {
		fmt.Println("Error: URL is required")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !*quiet {
		printBanner()
	}

	// Настраиваем сканер
	opts := []scanner.Option{
		scanner.WithTimeout(time.Duration(*timeout) * time.Second),
		scanner.WithWorkers(*workers),
		scanner.WithScanDepth(*depth),
		scanner.WithUserAgent("GoBruteScanner-CLI/1.0"),
	}

	// Добавляем прокси если есть
	if *proxies != "" {
		proxyURLs := loadLinesFromFile(*proxies)
		if len(proxyURLs) > 0 {
			opts = append(opts, scanner.WithProxyURLs(proxyURLs...))
			opts = append(opts, scanner.WithProxies(true))
			if !*quiet {
				fmt.Printf("[*] Loaded %d proxies\n", len(proxyURLs))
			}
		}
	}

	// Создаем сканер
	s, err := scanner.New(*url, opts...)
	if err != nil {
		fmt.Printf("❌ Failed to create scanner: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	var allResults []types.ScanResult

	// Фаза 1: Автообнаружение
	if *discover {
		if !*quiet {
			fmt.Println("\n[1/2] 🔍 Auto-discovery phase")
		}

		endpoints, err := s.Discover(ctx)
		if err != nil && !*quiet {
			fmt.Printf("⚠️ Discovery error: %v\n", err)
		}

		if !*quiet {
			fmt.Printf("   Discovered %d endpoints\n", len(endpoints))
		}
	}

	// Фаза 2: Брутфорс
	if *brute {
		if !*quiet {
			fmt.Println("\n[2/2] ⚡ Brute force phase")
		}

		// Загружаем словарь
		var wordlistItems []string
		if *wordlist != "" {
			wordlistItems = loadLinesFromFile(*wordlist)
			if !*quiet {
				fmt.Printf("   Loaded %d words from custom wordlist\n", len(wordlistItems))
			}
		} else {
			wl := wordlists.New()
			wordlistItems = wl.GetAll()
			if !*quiet {
				fmt.Printf("   Using built-in wordlist (%d words)\n", len(wordlistItems))
			}
		}

		// Парсим методы
		methodList := strings.Split(*methods, ",")
		for i := range methodList {
			methodList[i] = strings.TrimSpace(strings.ToUpper(methodList[i]))
		}

		if !*quiet {
			fmt.Printf("   Testing with methods: %s\n", strings.Join(methodList, ", "))
			fmt.Printf("   Workers: %d, Delay: %dms\n", *workers, *delay)
			fmt.Println("   Scanning...")
		}

		// Запускаем сканирование
		results, err := s.ScanWithWordlist(
			ctx,
			wordlistItems,
			methodList,
			*workers,
			time.Duration(*delay)*time.Millisecond,
		)
		if err != nil {
			fmt.Printf("❌ Scan failed: %v\n", err)
			os.Exit(1)
		}

		allResults = results

		if !*quiet {
			fmt.Printf("   Completed %d requests\n", len(results))
		}
	}

	// Анализ результатов
	if !*quiet {
		fmt.Println("\n📊 Results Analysis")
	}

	// Группируем по статус кодам
	statusCounts := make(map[int]int)
	var successful []types.ScanResult

	for _, result := range allResults {
		statusCounts[result.StatusCode]++
		if result.StatusCode >= 200 && result.StatusCode < 300 {
			successful = append(successful, result)
		}
	}

	// Выводим статистику
	if !*quiet {
		fmt.Println("\nStatus Code Summary:")
		for code, count := range statusCounts {
			emoji := "❓"
			switch {
			case code >= 200 && code < 300:
				emoji = "✅"
			case code >= 300 && code < 400:
				emoji = "↪️"
			case code >= 400 && code < 500:
				emoji = "🚫"
			case code >= 500:
				emoji = "🔥"
			}
			fmt.Printf("   %s %d: %d requests\n", emoji, code, count)
		}

		fmt.Printf("\n✅ Successful endpoints (%d):\n", len(successful))
		for i, result := range successful {
			if i >= 20 {
				fmt.Printf("   ... and %d more\n", len(successful)-20)
				break
			}
			title := result.Title
			if title == "" {
				title = "No title"
			}
			fmt.Printf("   • [%d] %s %s (%d bytes)\n",
				result.StatusCode, result.Method, result.URL, result.Size)
		}
	} else {
		// Quiet mode - только список успешных эндпоинтов
		for _, result := range successful {
			fmt.Printf("%s %s [%d]\n", result.Method, result.URL, result.StatusCode)
		}
	}

	// Экспорт результатов
	if *outputFile != "" {
		exportResults(allResults, *outputFile, *format)
		if !*quiet {
			fmt.Printf("\n💾 Results exported to %s (%s format)\n", *outputFile, *format)
		}
	}

	// Итоговая статистика
	if !*quiet {
		stats := s.GetStats()
		fmt.Println("\n📈 Final Statistics:")
		fmt.Printf("   • Total requests: %d\n", stats.TotalRequests)
		fmt.Printf("   • Successful (2xx): %d\n", stats.Successful)
		fmt.Printf("   • Failed (4xx/5xx): %d\n", stats.Failed)
		fmt.Printf("   • Total time: %v\n", stats.Duration)
		fmt.Printf("   • Requests/sec: %.1f\n",
			float64(stats.TotalRequests)/stats.Duration.Seconds())

		fmt.Println("\n🎉 Scan completed successfully!")
	}
}

// printBanner выводит баннер
func printBanner() {
	fmt.Println(`
╔══════════════════════════════════════════╗
║         Go Brute Scanner v1.0           ║
║      Automated API Endpoint Discovery    ║
╚══════════════════════════════════════════╝`)
	fmt.Println()
}

// loadLinesFromFile загружает строки из файла
func loadLinesFromFile(filename string) []string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			result = append(result, line)
		}
	}
	return result
}

// exportResults экспортирует результаты в файл
func exportResults(results []types.ScanResult, filename, format string) {
	var formatter output.Formatter
	var data string
	var err error

	// Преобразуем результаты в interface{}
	var interfaceResults []interface{}
	for _, r := range results {
		interfaceResults = append(interfaceResults, r)
	}

	// Выбираем форматтер
	switch strings.ToLower(format) {
	case "json":
		formatter = &output.JSONFormatter{Pretty: true}
	case "md", "markdown":
		formatter = &output.MarkdownFormatter{}
	case "txt", "text":
		formatter = &output.SimpleFormatter{}
	default:
		formatter = &output.JSONFormatter{Pretty: true}
	}

	data, err = formatter.Format(interfaceResults)
	if err != nil {
		fmt.Printf("Warning: Failed to format results: %v\n", err)
		return
	}

	// Сохраняем в файл
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Warning: Failed to create file: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(data)
}
