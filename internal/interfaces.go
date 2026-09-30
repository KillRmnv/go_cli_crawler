package clicrawler

import (
	"net/url"
	"golang.org/x/net/html"
)

type Storage interface {
	Save(data []ResourseNode) ([]byte,error)
}

type Parser interface {
 	ParsePage(page *string, domainUrl *string, pageUrl* string) (ResourseNode, []string)
  	ExtractPageData(doc *html.Node, baseURL *url.URL, domain string) (string, []string)
}
