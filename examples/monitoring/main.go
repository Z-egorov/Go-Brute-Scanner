package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Z-egorov/Go-Brute-Scanner/pkg/scanner"
	"github.com/Z-egorov/Go-Brute-Scanner/pkg/types"
)

func main() {
	fmt.Println("=== Go Brute Scanner - Continuous Monitoring ===")
	fmt.Println("Monitoring API endpoints for real-time updates...")

	// Целевые эндпоинты для мониторинга
	endpointsToMonitor := []string{
		"/api/health",
		"/api/status",
		"/api/metrics",
		"/api/coefficient", // Пример ручки с обновлениями в реальном времени
	}

	// Создаем сканер
	s, err := scanner.New(
		"http://localhost:8080",
		scanner.WithTimeout(5*time.Second),
		scanner.WithWorkers(3),
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Начальное сканирование
	fmt.Println("\nInitial scan:")
	initialResults := make(map[string]types.ScanResult)

	for _, endpoint := range endpointsToMonitor {
		results, err := s.ScanWithWordlist(
			ctx,
			[]string{endpoint},
			[]string{"GET"},
			1,
			0,
		)
		if err == nil && len(results) > 0 {
			initialResults[endpoint] = results[0]
			status := results[0].StatusCode
			size := results[0].Size
			fmt.Printf("  %s: %d (%d bytes)\n", endpoint, status, size)
		}
	}

	// Цикл мониторинга
	fmt.Println("\nStarting continuous monitoring (Ctrl+C to stop)...")
	fmt.Println("Press Enter to stop...")

	ticker := time.NewTicker(30 * time.Second) // Проверка каждые 30 секунд
	defer ticker.Stop()

	stop := make(chan bool)

	// Горутина для обработки ввода
	go func() {
		fmt.Scanln()
		stop <- true
	}()

monitoringLoop:
	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n[%s] Checking endpoints...\n", time.Now().Format("15:04:05"))

			for _, endpoint := range endpointsToMonitor {
				results, err := s.ScanWithWordlist(
					ctx,
					[]string{endpoint},
					[]string{"GET"},
					1,
					0,
				)

				if err != nil {
					fmt.Printf("  ❌ %s: Error - %v\n", endpoint, err)
					continue
				}

				if len(results) > 0 {
					current := results[0]

					// Проверяем изменения
					if initial, exists := initialResults[endpoint]; exists {
						if current.StatusCode != initial.StatusCode {
							fmt.Printf("  ⚠️ %s: Status changed %d → %d\n",
								endpoint, initial.StatusCode, current.StatusCode)
							initialResults[endpoint] = current
						}

						if current.Size != initial.Size {
							diff := current.Size - initial.Size
							fmt.Printf("  📊 %s: Size changed %d → %d (Δ%d)\n",
								endpoint, initial.Size, current.Size, diff)
							initialResults[endpoint] = current
						}

						// Для коэффициента парсим JSON
						if endpoint == "/api/coefficient" && current.Size > 0 {
							// Здесь можно добавить парсинг JSON и проверку значения
							fmt.Printf("  🔄 %s: Updated (%d bytes)\n", endpoint, current.Size)
						}
					} else {
						initialResults[endpoint] = current
						fmt.Printf("  ✅ %s: New endpoint found\n", endpoint)
					}
				}
			}

		case <-stop:
			fmt.Println("\nStopping monitoring...")
			break monitoringLoop
		}
	}

	fmt.Println("\nMonitoring stopped.")
	fmt.Println("Final status:")

	for endpoint, result := range initialResults {
		fmt.Printf("  %s: %d (%d bytes)\n", endpoint, result.StatusCode, result.Size)
	}
}
