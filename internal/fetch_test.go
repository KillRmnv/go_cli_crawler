package clicrawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestExtractErrStatusLogFilepath(t *testing.T) {
	client := &FetchClient{}

	tests := []struct {
		name     string
		input    string 
		expected string
	}{
		{"Just a file", "app.log", "app_error.log"},
		{"Filepath positive test", "/var/logs/my_app.txt", "/var/logs/my_app_error.txt"},
		{"File without extension", "syslog", "syslog_error.log"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := client.extractErrStatusLogFilepath(tc.input)

			if result != tc.expected {
				t.Error("Expected:"+tc.expected +", result:"+ result)
			}
		})
	}
}

func TestFetchPage(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "log_test", "crawler.log")
	outFile := filepath.Join(tempDir, "out_test", "result.json")
	
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/image") {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if strings.Contains(r.URL.Path, "/delay") {
			time.Sleep(100 * time.Millisecond)
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Mock Page</h1><p>This string is deliberately made longer than 100 characters to pass the length check in the positive fetch test. Here is some padding text to make it longer.</p></body></html>`))
	}))
	defer mockServer.Close()

	var mockUrls []string
	for i := 0; i < 15; i++ {
		mockUrls = append(mockUrls, fmt.Sprintf("%s/delay/%d", mockServer.URL, i))
	}

	var crawler CliCrawler
	var config CrawlerConfig
	config.SetDelay(0)
	config.SetDepth(1)
	config.SetLog(logFile)
	config.SetOutput(outFile)
	config.SetRequestTimeout(5 * time.Second)
	config.SetRetry(3)
	config.SetTimeout(10 * time.Second)
	config.SetUrlsSlice(mockUrls) 

	crawler.Init(&config)
	
	client := &FetchClient{}
	client.Init(&config)

	t.Run("Positive fetch", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := client.FetchPage(ctx, mockServer.URL+"/index", &crawler)
		if len(result) < 100 {
			t.Error("Does not get response: " + result)
		}
	})

	t.Run("File fetch", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := client.FetchPage(ctx, mockServer.URL+"/image.jpg", &crawler)
		if len(result) > 0 {
			t.Error("Expected empty string, but returned: " + result)
		}
	})

	t.Run("Gracefull shoutdown test", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup

		for i := 0; i < len(config.urls); i++ {
			wg.Add(1)

			go func(url string) {
				defer wg.Done()
				client.FetchPage(ctx, url, &crawler)
			}(config.urls[i].adress)

			if i == len(config.urls)-2 {
				cancel()
			}
		}

		wg.Wait()
		file, _ := os.ReadFile(config.log)
		if !strings.Contains(string(file), "Gracefully stopping waiting") && !strings.Contains(string(file), "Gracefully stopping fetching") {
			t.Error("Log file does not contain gracefull shutdown log:\n" + string(file))
		}
	})

	t.Run("Semaphore test", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup

		for i := 0; i < len(config.urls); i++ {
			wg.Add(1)

			go func(url string) {
				defer wg.Done()
				client.FetchPage(ctx, url, &crawler)
			}(config.urls[i].adress)

			if i == len(config.urls)-2 {
				cancel()
			}
		}

		wg.Wait()
		file, _ := os.ReadFile(config.log)
		if !strings.Contains(string(file), "Gracefully stopping fetching (in queue)") {
			t.Error("Log file does not contain semaphore shutdown log:\n" + string(file))
		}
	})
}