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
		if Stubs != false {
			t.Errorf("stubs flag must be disabled by default, got %v", Stubs)
		}
		config := &CrawlerConfig{}
		
		config.SetStubs(true)
		
		if config.stubs != true {
			t.Errorf("Expected stubs enabled after SetStubs(true)")
		}
		config.SetStubs(false)
		if config.stubs != false {
			t.Errorf("Expected stubs disabled after SetStubs(false)")
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

func TestCrawlerConfig_SettersValidation(t *testing.T) {
	t.Run("SetRetry zero and negative fall back to default", func(t *testing.T) {
		for _, v := range []int{0, -3} {
			config := &CrawlerConfig{}
			config.SetRetry(v)
			if config.retry != Retry {
				t.Errorf("SetRetry(%d): expected default %d, recieved %d", v, Retry, config.retry)
			}
		}
	})

	t.Run("SetDepth negative falls back to default, zero stays", func(t *testing.T) {
		config := &CrawlerConfig{}
		config.SetDepth(-5)
		if config.depth != Depth {
			t.Errorf("SetDepth(-5): expected default %d, recieved %d", Depth, config.depth)
		}
		config.SetDepth(0)
		if config.depth != 0 {
			t.Errorf("SetDepth(0): expected 0 (seeds only), recieved %d", config.depth)
		}
	})

	t.Run("SetTimeout and SetRequestTimeout non-positive fall back to defaults", func(t *testing.T) {
		config := &CrawlerConfig{}
		config.SetTimeout(0)
		if config.timeout != Timeout {
			t.Errorf("SetTimeout(0): expected default %v, recieved %v", Timeout, config.timeout)
		}
		config.SetRequestTimeout(-time.Second)
		if config.requestTimeout != RequestTimeout {
			t.Errorf("SetRequestTimeout(-1s): expected default %v, recieved %v", RequestTimeout, config.requestTimeout)
		}
	})

	t.Run("SetDelay negative falls back to default, zero stays", func(t *testing.T) {
		config := &CrawlerConfig{}
		config.SetDelay(-time.Second)
		if config.delay != Delay {
			t.Errorf("SetDelay(-1s): expected default %v, recieved %v", Delay, config.delay)
		}
		config.SetDelay(0)
		if config.delay != 0 {
			t.Errorf("SetDelay(0): expected 0, recieved %v", config.delay)
		}
	})

	t.Run("SetOutput and SetLog blank fall back to defaults", func(t *testing.T) {
		for _, v := range []string{"", "   "} {
			config := &CrawlerConfig{}
			config.SetOutput(v)
			if config.output != OutputFilepath {
				t.Errorf("SetOutput(%q): expected default %q, recieved %q", v, OutputFilepath, config.output)
			}
			config.SetLog(v)
			if config.log != LogFilepath {
				t.Errorf("SetLog(%q): expected default %q, recieved %q", v, LogFilepath, config.log)
			}
		}
	})

	t.Run("SetUrls skips blanks", func(t *testing.T) {
		config := &CrawlerConfig{}
		config.SetUrls("")
		if len(config.urls) != 0 {
			t.Fatalf("SetUrls(\"\"): expected 0 urls, recieved %d", len(config.urls))
		}
		config.SetUrls(" , , https://a.com, ")
		if len(config.urls) != 1 {
			t.Fatalf("expected 1 URL after blank skip, recieved %d", len(config.urls))
		}
		sliceConfig := &CrawlerConfig{}
		sliceConfig.SetUrlsSlice([]string{"", "  ", "https://b.com"})
		if len(sliceConfig.urls) != 1 {
			t.Fatalf("expected 1 URL after blank skip, recieved %d", len(sliceConfig.urls))
		}
	})
}