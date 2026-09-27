package clicrawler

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"log"
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

func TestCliCrawler_ParsePage(t *testing.T) {
	crawler := &CliCrawler{}
	crawler.crawlerLogger = *log.New(io.Discard, "", 0)

	htmlPage := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Test title</title>
	</head>
	<body>
		<a href="https://example.com/page1">Link1 (same domain)</a>
		<a href="https://other-domain.com/page2">Link2 (another domain)</a>
		<a href='https://example.com/page3'>Link3 (same domain)</a>
	</body>
	</html>
	`
	domainUrl := "example.com"

	result := crawler.parsePage(&htmlPage, &domainUrl)

	expectedTitle := "Test title"
	if result.Title != expectedTitle {
		t.Errorf("Expected title %q, recieved %q", expectedTitle, result.Title)
	}
	if result.Resourse != domainUrl {
		t.Errorf("Expected resource %q, recieved %q", domainUrl, result.Resourse)
	}

	var parsedLinks []string
	for _, link := range result.Links {
		if link.Resourse != "" { 
			parsedLinks = append(parsedLinks, link.Resourse)
		}
	}

	if len(parsedLinks) != 2 {
		t.Fatalf("Expexted 2 links of domain %s, recieved %d", domainUrl, len(parsedLinks))
	}

	expectedLink1 := "https://example.com/page1"
	expectedLink2 := "https://example.com/page3"

	if parsedLinks[0] != expectedLink1 && parsedLinks[1] != expectedLink1 {
		t.Errorf("link %q are not in result", expectedLink1)
	}
	if parsedLinks[0] != expectedLink2 && parsedLinks[1] != expectedLink2 {
		t.Errorf("link %q are not in result", expectedLink2)
	}
}

func TestCliCrawler_Crawle_CancelContext(t *testing.T) {
	tempDir := t.TempDir()
	config := &CrawlerConfig{}
	config.SetLog(filepath.Join(tempDir, "log.txt"))
	config.SetOutput(filepath.Join(tempDir, "out.json"))
	
	config.SetTimeout(10 * time.Millisecond)
	config.SetUrlsSlice([]string{"https://example.com"})
	config.SetDepth(1)

	crawler := &CliCrawler{}
	crawler.Init(config)
	
	_, err := crawler.Crawle()
	if err != nil {
		t.Errorf("Unexpected timeout error, recieved: %v", err)
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

func TestCliCrawler_Crawle_OutputSurvivesInvalidUTF8(t *testing.T) {
	broken := []ResourseNode{{Resourse: "example.com", Title: "Google \xcc\xe0\xeb"}}

	if _, err := json.Marshal(broken); err == nil {
		t.Error("default Marshal must reject invalid UTF-8 — this is why Crawle passes jsontext.AllowInvalidUTF8")
	}

	data, err := json.Marshal(broken, jsontext.AllowInvalidUTF8(true))
	if err != nil {
		t.Fatalf("Marshal with AllowInvalidUTF8 failed: %v", err)
	}
	if err := json.Unmarshal(data, new(any)); err != nil {
		t.Errorf("output is not valid JSON: %v (%s)", err, data)
	}
	if !strings.ContainsRune(string(data), '�') {
		t.Errorf("invalid bytes must be mangled to U+FFFD, got %s", data)
	}
}

func TestSendNode(t *testing.T) {
	node := ResourseNode{Resourse: "example.com", Title: "Test"}

	t.Run("Live ctx with buffered channel", func(t *testing.T) {
		ch := make(chan ResourseNode, 1)
		if !sendNode(ch, node, context.Background()) {
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

		if !sendNode(ch, node, ctx) {
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
		go func() { done <- sendNode(ch, node, ctx) }()

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