package clicrawler

import (
	"testing"
	"time"
)

func TestCrawlerConfig_Setters(t *testing.T) {
	t.Run("SetDelay", func(t *testing.T) {
		config := &CrawlerConfig{}
		expected := 5 * time.Second
		
		config.SetDelay(expected)
		
		if config.delay != expected {
			t.Errorf("Expected delay %v, recieved %v", expected, config.delay)
		}
	})

	t.Run("SetRetry", func(t *testing.T) {
		config := &CrawlerConfig{}
		expected := 3
		
		config.SetRetry(expected)
		
		if config.retry != expected {
			t.Errorf("Expected retry %d, recieved %d", expected, config.retry)
		}
	})

	t.Run("SetDepth", func(t *testing.T) {
		config := &CrawlerConfig{}
		expected := 50
		
		config.SetDepth(expected)
		
		if config.depth != expected {
			t.Errorf("Expected depth %d, recieved %d", expected, config.depth)
		}
	})

	t.Run("SetTimeout", func(t *testing.T) {
		config := &CrawlerConfig{}
		expected := 3 * time.Minute
		
		config.SetTimeout(expected)
		
		if config.timeout != expected {
			t.Errorf("Expected timeout %v, recieved %v", expected, config.timeout)
		}
	})

	t.Run("SetRequestTimeout", func(t *testing.T) {
		config := &CrawlerConfig{}
		expected := 15 * time.Second
		
		config.SetRequestTimeout(expected)
		
		if config.requestTimeout != expected {
			t.Errorf("Expected requestTimeout %v, recieved %v", expected, config.requestTimeout)
		}
	})

	t.Run("SetOutput", func(t *testing.T) {
		config := &CrawlerConfig{}
		expected := "./test_out.json"
		
		config.SetOutput(expected)
		
		if config.output != expected {
			t.Errorf("Expected output %q, recieved %q", expected, config.output)
		}
	})

	t.Run("SetLog", func(t *testing.T) {
		config := &CrawlerConfig{}
		expected := "./test_log.log"
		
		config.SetLog(expected)
		
		if config.log != expected {
			t.Errorf("Expected log %q, recieved %q", expected, config.log)
		}
	})

	t.Run("SetStubs", func(t *testing.T) {
		if Stubs != true {
			t.Errorf("stubs flag must be enabled by default, got %v", Stubs)
		}
		config := &CrawlerConfig{}
		
		config.SetStubs(false)
		
		if config.stubs != false {
			t.Errorf("Expected stubs disabled after SetStubs(false)")
		}
		config.SetStubs(true)
		if config.stubs != true {
			t.Errorf("Expected stubs enabled after SetStubs(true)")
		}
	})

	t.Run("SetUrlsSlice", func(t *testing.T) {
		config := &CrawlerConfig{}
		urlsSlice := []string{"https://example.com", "https://test.com"}
		
		config.SetUrlsSlice(urlsSlice)
		
		if len(config.urls) != 2 {
			t.Fatalf("Expected 2 URL in slice, recieved %d", len(config.urls))
		}
	})

	t.Run("SetUrls (string parse)", func(t *testing.T) {
		config := &CrawlerConfig{}
		inputStr := "https://a.com,https://b.com,https://c.com"
		
		config.SetUrls(inputStr)
		
		if len(config.urls) != 3 {
			t.Fatalf("Expected 3 URL after string parse, recieved %d", len(config.urls))
		}
	})
	
	t.Run("SetUrls (one URL)", func(t *testing.T) {
		config := &CrawlerConfig{}
		inputStr := "https://single.com"
		
		config.SetUrls(inputStr)
		
		if len(config.urls) != 1 {
			t.Fatalf("ожидался 1 URL, получили %d", len(config.urls))
		}
	})
}