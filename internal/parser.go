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
	baseURL = parser.applyBaseHref(doc, baseURL)
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


func (parser* StandardHTMLParser) applyBaseHref(doc *html.Node, baseURL *url.URL) *url.URL{
	rawBase:= parser.findBaseHref(doc)
	if rawBase==""{
		return baseURL
	}
	parsedBase, err := url.Parse(strings.TrimSpace(rawBase))
	if err!=nil{
		parser.logger.Printf("Skipped invalid <base href> %q: %v\n", rawBase, err)
		return baseURL
	}
	resolvedBase:= baseURL.ResolveReference(parsedBase)
	resolvedBase.Fragment= ""
	resolvedBase.RawFragment= ""
	parser.logger.Printf("Using <base href> %q for page %s\n", resolvedBase.String(), baseURL.String())
	return resolvedBase
}

func (parser* StandardHTMLParser) findBaseHref(doc *html.Node) string{
	var found string
	var traverse func(*html.Node)
	traverse= func(n *html.Node){
		if found!=""{
			return
		}
		if n.Type== html.ElementNode && n.Data=="base"{
			for _, attr := range n.Attr{
				if attr.Key=="href"{
					found= attr.Val
					return
				}
			}
		}
		for c:= n.FirstChild; c!=nil && found==""; c=c.NextSibling{
			traverse(c)
		}
	}
	traverse(doc)
	return found
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

	if !hostInScope(resolvedURL.Host, targetDomain) {
		return "", false
	}

	return resolvedURL.String(), true
}
