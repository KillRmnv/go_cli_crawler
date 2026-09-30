package clicrawler

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)


func TestCliCrawler_Init(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "log", "crawler.log")
	outFile := filepath.Join(tempDir, "out", "result.json")

	config := &CrawlerConfig{}
	config.SetLog(logFile)
	config.SetOutput(outFile)
	config.SetDepth(5)
	config.SetTimeout(10 * time.Second)

	crawler := &CliCrawler{}
	crawler.Init(config)

	if crawler.visited == nil {
		t.Error("crawler.visited not initialized")
	}

	if crawler.amountOfGorutines <= 0 {
		t.Errorf("Expected amountOfGorutines > 0, recieved %d", crawler.amountOfGorutines)
	}

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file does not exists: %s", logFile)
	}
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Errorf("Out file does not exists: %s", outFile)
	}
}

func TestCliCrawler_CreateFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.json")
	crawler := &CliCrawler{}

	file1 := crawler.createFile(filePath)
	if file1 == nil {
		t.Fatal("File must not be nil")
	}
	file1.Close() 

	file2 := crawler.createFile(filePath)
	if file2 == nil {
		t.Fatal("Timestamp file must not be nil")
	}
	defer file2.Close()

	name2 := file2.Name()
	if name2 == filePath {
		t.Errorf("Filename does not change, expected additional timestamp. Recieved: %s", name2)
	}
	if !strings.HasSuffix(name2, ".json") {
		t.Errorf("expected extension .json, recieved name: %s", name2)
	}
}

func TestCliCrawler_Crawle_CancelContext(t *testing.T) {
	tempDir := t.TempDir()
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Slow</title></head><body></body></html>`))
	}))
	defer slowServer.Close()
	config := &CrawlerConfig{}
	config.SetLog(filepath.Join(tempDir, "log.txt"))
	config.SetOutput(filepath.Join(tempDir, "out.json"))

	config.SetTimeout(100 * time.Millisecond)
	config.SetRequestTimeout(5 * time.Second)
	config.SetRetry(1)
	config.SetUrlsSlice([]string{slowServer.URL})
	config.SetDepth(1)

	crawler := &CliCrawler{}
	crawler.Init(config)

	_, err := crawler.Crawle()
	if err != nil {
		t.Errorf("Timeout unwind must stay graceful, recieved: %v", err)
	}
}

func TestCliCrawler_ExtractUrlsDomains(t *testing.T) {
	tempDir := t.TempDir()
	config := &CrawlerConfig{}
	config.SetLog(filepath.Join(tempDir, "log.txt"))
	config.SetOutput(filepath.Join(tempDir, "out.json"))
	config.SetUrls("https://a.com, not-a-url, https://b.com/path")

	crawler := &CliCrawler{}
	crawler.Init(config)

	links, domains := crawler.extractUrlsDomains()
	if len(links) != 2 {
		t.Fatalf("expected 2 valid urls, recieved %d", len(links))
	}
	if domains[0] != "a.com" || domains[1] != "b.com" {
		t.Errorf("unexpected domains: %v", domains)
	}
}

func TestCliCrawler_Crawle_DepthLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Page A</title></head><body><a href="/b">to b</a></body></html>`))
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Page B</title></head><body></body></html>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	crawlSeed := func(t *testing.T, depth int) []ResourseNode {
		t.Helper()
		tempDir := t.TempDir()
		config := &CrawlerConfig{}
		config.SetLog(filepath.Join(tempDir, "log.txt"))
		config.SetOutput(filepath.Join(tempDir, "out.json"))
		config.SetTimeout(20 * time.Second)
		config.SetRequestTimeout(5 * time.Second)
		config.SetRetry(1)
		config.SetUrlsSlice([]string{server.URL + "/a"})
		config.SetDepth(depth)

		crawler := &CliCrawler{}
		crawler.Init(config)

		data, err := crawler.Crawle()
		if err != nil {
			t.Fatalf("unexpected crawle error: %v", err)
		}
		var nodes []ResourseNode
		if err := json.Unmarshal(data, &nodes); err != nil {
			t.Fatalf("result is not valid JSON: %v", err)
		}
		return nodes
	}

	t.Run("Depth zero fetches seeds only", func(t *testing.T) {
		nodes := crawlSeed(t, 0)
		if len(nodes) != 1 {
			t.Fatalf("expected 1 root, recieved %d", len(nodes))
		}
		if nodes[0].Title != "Page A" {
			t.Errorf("expected title %q, recieved %q", "Page A", nodes[0].Title)
		}
		if len(nodes[0].Links) != 0 {
			t.Errorf("depth 0 must not fetch children, recieved %d links", len(nodes[0].Links))
		}
	})

	t.Run("Depth one fetches children", func(t *testing.T) {
		nodes := crawlSeed(t, 1)
		if len(nodes) != 1 {
			t.Fatalf("expected 1 root, recieved %d", len(nodes))
		}
		if len(nodes[0].Links) != 1 {
			t.Fatalf("expected 1 child, recieved %d", len(nodes[0].Links))
		}
		child := nodes[0].Links[0]
		if child.Title != "Page B" {
			t.Errorf("expected child title %q, recieved %q", "Page B", child.Title)
		}
		if !strings.HasSuffix(child.Resourse, "/b") {
			t.Errorf("expected child resource ending with /b, recieved %q", child.Resourse)
		}
	})
}

func TestCliCrawler_Crawle_NoValidUrls(t *testing.T) {
	tempDir := t.TempDir()
	outFile := filepath.Join(tempDir, "out.json")
	config := &CrawlerConfig{}
	config.SetLog(filepath.Join(tempDir, "log.txt"))
	config.SetOutput(outFile)
	config.SetTimeout(5 * time.Second)
	config.SetUrls("not-a-url, ,")

	crawler := &CliCrawler{}
	crawler.Init(config)

	_, err := crawler.Crawle()
	if err == nil {
		t.Fatal("Crawle must fail when no seed url is valid")
	}
	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("empty result must still be saved: %v", readErr)
	}
	if strings.TrimSpace(string(data)) != "[]" {
		t.Errorf("expected empty array in output, recieved %q", data)
	}
}

func TestCliCrawler_Crawle_AllSeedsFail(t *testing.T) {
	tempDir := t.TempDir()
	deadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadUrl := deadServer.URL
	deadServer.Close()
	config := &CrawlerConfig{}
	config.SetLog(filepath.Join(tempDir, "log.txt"))
	config.SetOutput(filepath.Join(tempDir, "out.json"))
	config.SetTimeout(20 * time.Second)
	config.SetRequestTimeout(2 * time.Second)
	config.SetRetry(1)
	config.SetUrlsSlice([]string{deadUrl})
	config.SetDepth(1)

	crawler := &CliCrawler{}
	crawler.Init(config)

	_, err := crawler.Crawle()
	if err == nil {
		t.Fatal("Crawle must fail when every seed fetch failed")
	}
}

func TestCliCrawler_Init_KeepsConfigAndRenamesOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "log", "crawler.log")
	outFile := filepath.Join(tempDir, "out", "result.json")
	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outFile, []byte("old result"), 0644); err != nil {
		t.Fatal(err)
	}

	config := &CrawlerConfig{}
	config.SetLog(logFile)
	config.SetOutput(outFile)
	config.SetDepth(7)

	crawler := &CliCrawler{}
	crawler.Init(config)

	if crawler.config.depth != 7 {
		t.Errorf("Init must keep config values, recieved depth=%d", crawler.config.depth)
	}
	if crawler.config.output == outFile {
		t.Errorf("output must be renamed when %s already exists, got plain path", outFile)
	}
	if _, err := os.Stat(crawler.config.output); err != nil {
		t.Errorf("renamed output file was not created: %v", err)
	}
	if data, _ := os.ReadFile(outFile); string(data) != "old result" {
		t.Errorf("previous result must stay untouched, got %q", data)
	}
}

func TestSendNode(t *testing.T) {
	node := ResourseNode{Resourse: "example.com", Title: "Test"}
	crawler := &CliCrawler{}
	t.Run("Live ctx with buffered channel", func(t *testing.T) {
		ch := make(chan ResourseNode, 1)
		crawler := &CliCrawler{}
		if !crawler.sendNode(ch, node, context.Background()) {
			t.Fatal("sendNode must deliver when channel has space")
		}
		if got := <-ch; got.Title != node.Title {
			t.Errorf("got %q, want %q", got.Title, node.Title)
		}
	})

	t.Run("Cancelled ctx with waiting reader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ch := make(chan ResourseNode)
		received := make(chan ResourseNode, 1)
		go func() { received <- <-ch }()
		time.Sleep(10 * time.Millisecond)

		if !crawler.sendNode(ch, node, ctx) {
			t.Fatal("node must be delivered to a waiting reader even when ctx is done")
		}
		if got := <-received; got.Resourse != node.Resourse {
			t.Errorf("got %q, want %q", got.Resourse, node.Resourse)
		}
	})

	t.Run("Cancelled ctx without reader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ch := make(chan ResourseNode)
		done := make(chan bool, 1)
		go func() { done <- crawler.sendNode(ch, node, ctx) }()

		select {
		case ok := <-done:
			if ok {
				t.Error("sendNode must report drop when blocked and ctx is done")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("sendNode hung without reader")
		}
	})
}