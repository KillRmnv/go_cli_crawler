package clicrawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestFetchClient_Retry(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "log_test", "crawler.log")
	outFile := filepath.Join(tempDir, "out_test", "result.json")

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			if hj, ok := w.(http.Hijacker); ok {
				if conn, _, err := hj.Hijack(); err == nil {
					conn.Close()
				}
			}
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Retry Page</title></head><body>This body is deliberately long enough to prove the page was really fetched after the first attempt failed at transport level.</body></html>`))
	}))
	defer server.Close()

	var crawler CliCrawler
	var config CrawlerConfig
	config.SetDelay(0)
	config.SetDepth(1)
	config.SetLog(logFile)
	config.SetOutput(outFile)
	config.SetRequestTimeout(5 * time.Second)
	config.SetRetry(1)
	config.SetTimeout(20 * time.Second)
	config.SetUrlsSlice([]string{server.URL})

	crawler.Init(&config)

	client := &FetchClient{}
	client.Init(&config)

	t.Run("First failure is retried", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := client.FetchPage(ctx, server.URL, &crawler)
		if len(result) == 0 {
			t.Fatal("expected page body after retry, recieved empty string")
		}
		if !strings.Contains(result, "Retry Page") {
			t.Errorf("unexpected body: %q", result)
		}
		if got := attempts.Load(); got < 2 {
			t.Errorf("expected at least one retry, total attempts=%d", got)
		}
	})

	t.Run("Persistent failure returns empty", func(t *testing.T) {
		deadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		deadUrl := deadServer.URL
		deadServer.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if result := client.FetchPage(ctx, deadUrl, &crawler); result != "" {
			t.Errorf("expected empty string on persistent failure, recieved %q", result)
		}
	})
}

func newFetchFixture(t *testing.T, configure func(*CrawlerConfig)) (*CliCrawler, *FetchClient, CrawlerConfig) {
	t.Helper()
	tempDir := t.TempDir()
	var config CrawlerConfig
	config.SetDelay(0)
	config.SetDepth(1)
	config.SetLog(filepath.Join(tempDir, "log_test", "crawler.log"))
	config.SetOutput(filepath.Join(tempDir, "out_test", "result.json"))
	config.SetRequestTimeout(5 * time.Second)
	config.SetRetry(3)
	config.SetTimeout(10 * time.Second)
	if configure != nil {
		configure(&config)
	}
	var crawler CliCrawler
	crawler.Init(&config)
	client := &FetchClient{}
	client.Init(&config)
	return &crawler, client, config
}

func waitForLog(t *testing.T, path, substr string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), substr) {
			return string(data)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %q in %s", substr, path)
			return ""
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestFetchPage(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/image") {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Mock Page</h1><p>This string is deliberately made longer than 100 characters to pass the length check in the positive fetch test. Here is some padding text to make it longer.</p></body></html>`))
	}))
	defer mockServer.Close()

	t.Run("Positive fetch", func(t *testing.T) {
		crawler, client, _ := newFetchFixture(t, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := client.FetchPage(ctx, mockServer.URL+"/index", crawler)
		if len(result) < 100 {	

			t.Error("Does not get response: " + result)
		}
	})

	t.Run("File fetch", func(t *testing.T) {
		crawler, client, _ := newFetchFixture(t, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := client.FetchPage(ctx, mockServer.URL+"/image.jpg", crawler)
		if len(result) > 0 {
			t.Error("Expected empty string, but returned: " + result)
		}
	})

	t.Run("Cancelled while waiting for semaphore", func(t *testing.T) {
		crawler, client, config := newFetchFixture(t, nil)

		for i := 0; i < cap(client.semaphore); i++ {
			client.semaphore <- struct{}{}
		}
		defer func() {
			for i := 0; i < cap(client.semaphore); i++ {
				<-client.semaphore
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan string, 1)
		go func() {
			done <- client.FetchPage(ctx, mockServer.URL+"/index", crawler)
		}()

		waitForLog(t, config.log, "Trying to fetch:", 5*time.Second)
		cancel()

		select {
		case result := <-done:
			if result != "" {
				t.Error("Expected empty string on cancel, but returned: " + result)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("FetchPage did not return after cancel")
		}

		file, _ := os.ReadFile(config.log)
		if !strings.Contains(string(file), "Gracefully stopping fetching (in queue)") {
			t.Error("Log file does not contain semaphore shutdown log:\n" + string(file))
		}
	})

	t.Run("Cancelled while waiting between retries", func(t *testing.T) {
		deadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		deadUrl := deadServer.URL
		deadServer.Close()

		crawler, client, config := newFetchFixture(t, func(c *CrawlerConfig) {
			c.SetDelay(30 * time.Second)
			c.SetRetry(3)
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan string, 1)
		go func() {
			done <- client.FetchPage(ctx, deadUrl, crawler)
		}()

		waitForLog(t, client.extractErrStatusLogFilepath(config.log), "Error while GET request", 5*time.Second)
		cancel()

		select {
		case result := <-done:
			if result != "" {
				t.Error("Expected empty string on cancel, but returned: " + result)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("FetchPage did not return after cancel")
		}

		file, _ := os.ReadFile(config.log)
		if !strings.Contains(string(file), "Gracefully stopping waiting") {
			t.Error("Log file does not contain graceful shutdown log:\n" + string(file))
		}
	})
}