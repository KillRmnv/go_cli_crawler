package clicrawler

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
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

func TestFetchPage(t *testing.T){
	client := &FetchClient{}
	ctx,cancel:=context.WithCancel(context.Background())
	var crawler CliCrawler
	var config CrawlerConfig
	config.SetDelay(Delay)
	config.SetDepth(Depth)
	config.SetLog("./log_test/log.log")
	config.SetOutput("./out_test/out.json")
	config.SetRequestTimeout(RequestTimeout)
	config.SetRetry(3)
	config.SetTimeout(Timeout)
	config.SetUrlsSlice([]string{
    
    "https://example.com",
    "https://example.org",
    "https://example.net",
    "https://go.dev",
    "https://pkg.go.dev",
    "https://en.wikipedia.org/wiki/Go_(programming_language)",
    "https://en.wikipedia.org/wiki/Concurrency_(computer_science)",
    "https://www.w3.org/",
    "https://developer.mozilla.org/en-US/",
    "https://github.com",
	
    "https://httpbin.org/delay/2",
    "https://httpbin.org/delay/2?req=1",
    "https://httpbin.org/delay/2?req=2",
    
    "https://httpbin.org/html", 
    "https://httpbin.org/xml",  
	})
	crawler.Init(&config)
	client.Init(&config)
	t.Run("Positive fetch", func(t *testing.T) {
		result := client.FetchPage(ctx,"https://google.com",&crawler)
		if len(result)<100 {
			t.Error("Does not get response:"+ result)
			
		}
	})
	t.Run("File fetch", func(t *testing.T) {
		result := client.FetchPage(ctx,"https://miro.medium.com/v2/resize:fit:720/format:webp/1*Xj9o_nJ7GJalHI60HtK3Kg.jpeg",&crawler)
		if len(result)>0 {
			t.Error("Expected empty string, but returned:"+ result)
		}
	})

	
	t.Run("Gracefull shoutdown test", func(t *testing.T) {
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
		file,_:=os.ReadFile(config.log)
		if(!strings.Contains(string(file),"Gracefully stopping waiting")&&!strings.Contains(string(file),"Gracefully stopping fetching")){
			t.Error("Log file does not contain grscefull shoutdown log:"+ string(file))
		}
	})
	t.Run("Semaphore test", func(t *testing.T) {
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
		file,_:=os.ReadFile(config.log)
		if(!strings.Contains(string(file),"Gracefully stopping fetching (in queue)")){
			t.Error("Log file does not contain semaphore shoutdown log:"+ string(file))
		}
	})
	
}

