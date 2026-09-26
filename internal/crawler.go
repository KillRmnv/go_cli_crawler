package clicrawler

import (
	"context"
	"encoding/json/v2"
	"log"
	"os"
	"regexp"
	"strconv"
	"github.com/deckarep/golang-set/v2"
)
type CliCrawler struct{
	config CrawlerConfig
	visited mapset.Set[string]
	crawlerLogger log.Logger
}

func (crawler* CliCrawler) Init(config CrawlerConfig){
	file, _ := os.OpenFile(config.log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	crawler.crawlerLogger=*log.New(file, "[CRAWLER] ", log.Lshortfile)
}
func (crawler* CliCrawler) crawle() ([]byte,error){
	result:=make([]ResourseNode,10)
	ctx, cancel := context.WithTimeout(context.Background(), crawler.config.timeout)
	defer cancel()

	resourseChans:=make(chan ResourseNode,10)
	for i:=0;i<len(crawler.config.urls);i++ {
		go	crawler.crawleInside(0,ctx,crawler.config.urls[i],resourseChans)
	}
	for resource := range resourseChans {
    	result = append(result, resource)
	}
	
	return json.Marshal(result)
}

func (crawler* CliCrawler) crawleInside(depth int,ctx context.Context,url string, resources chan ResourseNode) {
	depth++
	if(depth>crawler.config.depth){
		// exit 
	}
	page:=fetchPage(ctx,url,&crawler.visited)
	var resourceNode ResourseNode
	resourceNode=parsePage(&page,crawler)
	resourseChans:=make(chan ResourseNode,10)
	for _,v:=range resourceNode.Links{
		if(!crawler.visited.Contains(v.Resourse)){
			crawler.crawlerLogger.Println("["+resourceNode.Resourse+"]"+"Crawling to "+v.Resourse)	
		    go	crawler.crawleInside(depth,ctx,v.Resourse,resourseChans)
		}else{
			crawler.crawlerLogger.Println("["+resourceNode.Resourse+"]"+"Found already visited link "+v.Resourse)
		}
	}
	for resource := range resourseChans {
    	resourceNode.Links = append(resourceNode.Links , resource)
	}
	
	resources<-resourceNode
}


func parsePage(page* string, crawler* CliCrawler) ResourseNode{
	var resorce ResourseNode
	reTitle:=regexp.MustCompile("<title>.*</title>")
	title:=reTitle.FindString(*page)
	resorce.Resourse=title[7:len(title)-8]
	reHref := regexp.MustCompile(`href=\"[a-zA-Z0-9\./]+\"`)
	hrefs:=reHref.FindAll([]byte(*page),-1)
	crawler.crawlerLogger.Println(2,"Found links on page "+resorce.Resourse+":"+strconv.Itoa(len(hrefs)))
	resorce.Links=make([]ResourseNode,len(hrefs))
	for i,href:= range hrefs{
		crawler.crawlerLogger.Println(2,resorce.Resourse+":"+string(href))
		resorce.Links[i].Resourse= string(href[6 : len(href)-1])
	}
	return resorce
}
