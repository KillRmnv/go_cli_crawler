package clicrawler

import (
	"encoding/json/v2"

	"github.com/deckarep/golang-set/v2"
)
type CliCrawler struct{
	config CrawlerConfig
}

func (crawler* CliCrawler) crawle() ([]byte,error){
	result:=make([]ResourseNode,10)
	for _,v:=range crawler.config.urls {
		visitedUrls:= mapset.NewSet[string]() // потокобезопасное множество
		result=append(result,crawler.crawleInside(v,&visitedUrls,0))
	}
	return json.Marshal(result)
}
func (crawler* CliCrawler) crawleInside(url string,visited* mapset.Set[string],depth int ) ResourseNode{
	
}