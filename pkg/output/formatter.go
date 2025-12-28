package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Formatter interface for format output
type Formatter interface {
	Format(results []interface{}) (string, error)
}

// JSONFormatter output in json
type JSONFormatter struct {
	Pretty bool
}

func (j *JSONFormatter) Format(results []interface{}) (string, error) {
	if j.Pretty {
		data, err := json.MarshalIndent(results, "", "  ")
		return string(data), err
	}
	data, err := json.Marshal(results)
	return string(data), err
}

// MarkdownFormatter output in markdown
type MarkdownFormatter struct{}

func (m *MarkdownFormatter) Format(results []interface{}) (string, error) {
	var sb strings.Builder

	sb.WriteString("# API Endpoints Discovery\n\n")
	sb.WriteString("| Method | URL | Status | Size | Title |\n")
	sb.WriteString("|--------|-----|--------|------|-------|\n")

	for _, result := range results {
		if r, ok := result.(map[string]interface{}); ok {
			statusCode, _ := r["status_code"].(float64)
			if statusCode >= 200 && statusCode < 400 {
				method, _ := r["method"].(string)
				url, _ := r["url"].(string)
				size, _ := r["size"].(float64)
				title, _ := r["title"].(string)

				sb.WriteString(fmt.Sprintf("| %s | `%s` | %.0f | %.0f | %s |\n",
					method, url, statusCode, size, title))
			}
		}
	}

	return sb.String(), nil
}

// SimpleFormatter simple output
type SimpleFormatter struct{}

func (s *SimpleFormatter) Format(results []interface{}) (string, error) {
	var sb strings.Builder

	for _, result := range results {
		if r, ok := result.(map[string]interface{}); ok {
			method, _ := r["method"].(string)
			url, _ := r["url"].(string)
			statusCode, _ := r["status_code"].(float64)

			sb.WriteString(fmt.Sprintf("%s %s - %d\n", method, url, int(statusCode)))
		}
	}

	return sb.String(), nil
}
