package clicrawler

import (
	"log"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Parser interface {
 	ParsePage(page *string, domainUrl *string, pageUrl* string) (ResourseNode, []string)
   	ExtractPageData(doc *html.Node, baseURL *url.URL, domain string) (string, []string)
}

type StandardHTMLParser struct{
	logger log.Logger
	iterateStubs bool
}


func (parser* StandardHTMLParser) ParsePage(page *string, domainUrl *string, pageUrl* string) (ResourseNode, []string) {
	var resource ResourseNode
	resource.Resourse = *pageUrl
	
	baseURL, err := url.Parse(*pageUrl)
	if err != nil {
		parser.logger.Println("Error parsing base URL:"+ err.Error())
		return resource, nil
	}
	
	doc, err := html.Parse(strings.NewReader(*page))
	if err != nil {
		parser.logger.Println("Error parsing HTML:"+ err.Error())
		return resource, nil
	}
	parser.logger.Printf("Successfully parsed:%s",*pageUrl)
	title, hrefsToCrawl := parser.ExtractPageData(doc, baseURL, *domainUrl)
	resource.Title = title

	var links []ResourseNode
	if parser.iterateStubs {
		for _, href := range hrefsToCrawl {
			links = append(links, ResourseNode{Resourse: href})
		}
	}
	resource.Links = links

	parser.logger.Printf("Found links on page %s: %d\n", resource.Title, len(hrefsToCrawl))
	
	return resource, hrefsToCrawl
}

func  (parser* StandardHTMLParser)ExtractPageData(doc *html.Node, baseURL *url.URL, domain string) (string, []string) {
	var title string
	var hrefs []string

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && title == "" {
					var titleBuilder strings.Builder
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							titleBuilder.WriteString(c.Data)
						}
					}
					title = strings.TrimSpace(titleBuilder.String())
				}

		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					if cleanURL, valid := parser.processHref(attr.Val, baseURL, domain); valid {
						hrefs = append(hrefs, cleanURL)
					}
					break
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return title, hrefs
}

func (parser *StandardHTMLParser) processHref(rawHref string, baseURL *url.URL, targetDomain string) (string, bool) {
	rawHref = strings.TrimSpace(rawHref)

	hrefURL, err := url.Parse(rawHref)
	if err != nil {
		parser.logger.Printf("Skipped invalid URL %q: %v\n", rawHref, err)
		return "", false
	}

	resolvedURL := baseURL.ResolveReference(hrefURL)
	resolvedURL.Fragment = ""

	if !strings.Contains(resolvedURL.Host, targetDomain) {
		return "", false
	}

	return resolvedURL.String(), true
}
