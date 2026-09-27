package clicrawler

import (
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
	if result.Resourse != expectedTitle {
		t.Errorf("Expected title %q, recieved %q", expectedTitle, result.Resourse)
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