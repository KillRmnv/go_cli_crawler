package clicrawler

import (
	"io"
	"log"
	"testing"
)


func TestParser_ParsePage(t *testing.T) {
	crawler := &CliCrawler{}
	crawler.parser=&StandardHTMLParser{logger: *log.New(io.Discard, "", log.LstdFlags|log.Lshortfile),iterateStubs:true}

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

	crawler.config.stubs = true
	result, hrefsToCrawl := crawler.parser.ParsePage(&htmlPage, &domainUrl,&domainUrl)

	expectedTitle := "Test title"
	if result.Title != expectedTitle {
		t.Errorf("Expected title %q, recieved %q", expectedTitle, result.Title)
	}
	if result.Resourse != domainUrl {
		t.Errorf("Expected resource %q, recieved %q", domainUrl, result.Resourse)
	}
	if len(hrefsToCrawl) != 2 {
		t.Errorf("crawl list must hold both domain links regardless of stubs flag, got %d", len(hrefsToCrawl))
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

func TestParser_ParsePage_StubsFlag(t *testing.T) {
	crawler := &CliCrawler{}
	crawler.crawlerLogger = *log.New(io.Discard, "", 0)

	htmlPage := `<title>T</title><a href="https://example.com/page1">a</a><a href="https://other.com/x">b</a>`
	domainUrl := "example.com"

	crawler.parser=&StandardHTMLParser{logger: *log.New(io.Discard, "", log.LstdFlags|log.Lshortfile),iterateStubs:true}
	node, hrefs := crawler.parser.ParsePage(&htmlPage, &domainUrl, &domainUrl)
	if len(hrefs) != 1 {
		t.Errorf("stubs=true: expected 1 crawl link, got %d", len(hrefs))
	}
	if len(node.Links) != 1 {
		t.Errorf("stubs=true: expected 1 stub in links, got %d", len(node.Links))
	}

	crawler.parser=&StandardHTMLParser{logger: *log.New(io.Discard, "", log.LstdFlags|log.Lshortfile),iterateStubs:false}
	node, hrefs = crawler.parser.ParsePage(&htmlPage, &domainUrl, &domainUrl)
	if len(hrefs) != 1 {
		t.Errorf("stubs=false: crawl list must not depend on the flag, got %d", len(hrefs))
	}
	if len(node.Links) != 0 {
		t.Errorf("stubs=false: links must contain no stubs, got %d", len(node.Links))
	}
}
func TestParser_RelativeAndSpecialLinks(t *testing.T) {
	parser := &StandardHTMLParser{
		logger:       *log.New(io.Discard, "", 0),
		iterateStubs: true,
	}

	pageUrl := "https://example.com/folder/subfolder/index.html"
	domainUrl := "example.com"

	htmlPage := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Special Links Test</title>
	</head>
	<body>
		<!-- 1. Абсолютный путь от корня домена -->
		<a href="/relative-root">Root</a>
		
		<!-- 2. Относительный путь (на уровень выше) -->
		<a href="../relative-up">Up</a>
		
		<!-- 3. Путь с параметрами запроса (query) -->
		<a href="page?query=123&sort=asc">Query</a>
		
		<!-- 4. Ссылка только на якорь (fragment) -->
		<a href="#section-2">Fragment</a>
		
		<!-- 5. Путь с якорем (якорь должен быть отброшен) -->
		<a href="/path#fragment">Path with frag</a>
	</body>
	</html>
	`

	_, hrefsToCrawl := parser.ParsePage(&htmlPage, &domainUrl, &pageUrl)

	expectedLinks := []string{
		"https://example.com/relative-root",                          
		"https://example.com/folder/relative-up",                     
		"https://example.com/folder/subfolder/page?query=123&sort=asc",
		"https://example.com/folder/subfolder/index.html",             
		"https://example.com/path",                                    
	}
	if len(hrefsToCrawl) != len(expectedLinks) {
		t.Fatalf("Expected %d links, got %d. Links: %v", len(expectedLinks), len(hrefsToCrawl), hrefsToCrawl)
	}

	for i, expected := range expectedLinks {
		if hrefsToCrawl[i] != expected {
			t.Errorf("Link mismatch at index %d.\nExpected: %q\nGot:      %q", i, expected, hrefsToCrawl[i])
		}
	}
}